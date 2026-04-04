package items

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/milagre/zote/go/zamqp"
	"github.com/milagre/zote/go/zapi"
	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zorm"

	"github.com/milagre/zote/examples/project/src/service/internal/api/names"
	"github.com/milagre/zote/examples/project/src/service/internal/api/resources"
	"github.com/milagre/zote/examples/project/src/service/internal/models"
	"github.com/milagre/zote/examples/project/src/service/internal/workers/deleter"
)

type item struct {
	repo      zorm.Repository
	publisher zamqp.Publisher
}

// newItem serves GET/PUT/DELETE /items/{item_id}.
func newItem(repo zorm.Repository, publisher zamqp.Publisher) zapi.Route {
	return &item{repo: repo, publisher: publisher}
}

func (r *item) Name() string { return names.ItemsItem }

func (r *item) Path() string {
	return zapi.Pathf("/items/%s", "item_id")
}

func (r *item) Methods() zapi.Methods {
	return zapi.Methods{
		http.MethodGet:    {Handler: r.get},
		http.MethodPut:    {Handler: r.put},
		http.MethodDelete: {Handler: r.delete},
	}
}

func (r *item) get(req zapi.Request) zapi.ResponseBuilder {
	ctx := req.Context()
	id := req.Param("item_id")
	if id == "" {
		return zapi.Response404NotFound()
	}

	row := &models.Item{ID: id}
	if err := zorm.Get(ctx, r.repo, []*models.Item{row}, zorm.GetOptions{}); err != nil {
		if errors.Is(err, zorm.ErrNotFound) {
			return zapi.Response404NotFound()
		}
		zlog.FromContext(ctx).Errorf("get item: %v", err)
		return zapi.Response500InternalServerError()
	}
	if row.Deleted {
		return zapi.Response404NotFound()
	}
	body, _ := json.Marshal(resources.ItemFromModel(row))
	return zapi.BasicResponse(http.StatusOK, jsonHeader(), body)
}

type putBody struct {
	Name string `json:"name"`
}

func (r *item) put(req zapi.Request) zapi.ResponseBuilder {
	ctx := req.Context()
	logger := zlog.FromContext(ctx)
	id := req.Param("item_id")
	if id == "" {
		return zapi.Response404NotFound()
	}

	raw, err := req.Body()
	if err != nil {
		return zapi.BasicResponse(http.StatusBadRequest, jsonHeader(), []byte(`{"error":"bad body"}`))
	}
	var pb putBody
	if err := json.Unmarshal(raw, &pb); err != nil || pb.Name == "" {
		return zapi.BasicResponse(http.StatusBadRequest, jsonHeader(), []byte(`{"error":"name required"}`))
	}

	row := &models.Item{ID: id}
	if err := zorm.Get(ctx, r.repo, []*models.Item{row}, zorm.GetOptions{}); err != nil {
		if errors.Is(err, zorm.ErrNotFound) {
			return zapi.Response404NotFound()
		}
		logger.Errorf("get item: %v", err)
		return zapi.Response500InternalServerError()
	}
	if row.Deleted {
		return zapi.BasicResponse(http.StatusConflict, jsonHeader(), []byte(`{"error":"item deleted"}`))
	}
	now := time.Now().UTC()
	row.Name = pb.Name
	row.Modified = &now
	if err := zorm.Put(ctx, r.repo, []*models.Item{row}, zorm.PutOptions{}); err != nil {
		logger.Errorf("put item: %v", err)
		return zapi.Response500InternalServerError()
	}
	body, _ := json.Marshal(resources.ItemFromModel(row))
	return zapi.BasicResponse(http.StatusOK, jsonHeader(), body)
}

func (r *item) delete(req zapi.Request) zapi.ResponseBuilder {
	ctx := req.Context()
	logger := zlog.FromContext(ctx)
	id := req.Param("item_id")
	if id == "" {
		return zapi.Response404NotFound()
	}

	row := &models.Item{ID: id}
	if err := zorm.Get(ctx, r.repo, []*models.Item{row}, zorm.GetOptions{}); err != nil {
		if errors.Is(err, zorm.ErrNotFound) {
			return zapi.Response404NotFound()
		}
		logger.Errorf("get item: %v", err)
		return zapi.Response500InternalServerError()
	}
	if row.Deleted {
		// idempotent: already queued / deleted
		return zapi.BasicResponse(http.StatusNoContent, http.Header{}, nil)
	}

	row.Deleted = true
	now := time.Now().UTC()
	row.Modified = &now
	if err := zorm.Put(ctx, r.repo, []*models.Item{row}, zorm.PutOptions{}); err != nil {
		logger.Errorf("soft delete: %v", err)
		return zapi.Response500InternalServerError()
	}

	if _, err := r.publisher.Publish(ctx, deleter.DeleteItemMessage{ItemID: id}); err != nil {
		logger.Errorf("publish delete: %v", err)
		return zapi.Response500InternalServerError()
	}
	return zapi.BasicResponse(http.StatusNoContent, http.Header{}, nil)
}
