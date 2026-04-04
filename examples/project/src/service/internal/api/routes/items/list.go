package items

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zapi"
	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zorm"

	"github.com/milagre/zote/examples/project/src/service/internal/api/names"
	"github.com/milagre/zote/examples/project/src/service/internal/api/resources"
	"github.com/milagre/zote/examples/project/src/service/internal/models"
	"github.com/milagre/zote/examples/project/src/service/internal/where"
)

type list struct {
	repo zorm.Repository
}

// newList serves GET /items and POST /items.
func newList(repo zorm.Repository) zapi.Route {
	return &list{repo: repo}
}

func (r *list) Name() string { return names.ItemsList }

func (r *list) Path() string { return "items" }

func (r *list) Methods() zapi.Methods {
	return zapi.Methods{
		http.MethodGet:  {Handler: r.get},
		http.MethodPost: {Handler: r.post},
	}
}

func (r *list) get(req zapi.Request) zapi.ResponseBuilder {
	ctx := req.Context()
	logger := zlog.FromContext(ctx)

	wclause, err := where.ParseWhere(req.Query("where"))
	if err != nil {
		logger.Warnf("where parse: %v", err)
		return zapi.BasicResponse(http.StatusBadRequest, jsonHeader(), []byte(`{"error":"invalid where"}`))
	}
	base := zelem.False(zelem.Field("Items.Deleted"))
	var full zorm.FindOptions
	if wclause != nil {
		full.Where = zelem.And(base, wclause)
	} else {
		full.Where = base
	}

	var rows []*models.Item
	if err := zorm.Find(ctx, r.repo, &rows, full); err != nil {
		logger.Errorf("find items: %v", err)
		return zapi.Response500InternalServerError()
	}
	body, err := json.Marshal(resources.ItemListFromModels(rows))
	if err != nil {
		return zapi.Response500InternalServerError()
	}
	return zapi.BasicResponse(http.StatusOK, jsonHeader(), body)
}

type postBody struct {
	Name string `json:"name"`
}

func (r *list) post(req zapi.Request) zapi.ResponseBuilder {
	ctx := req.Context()
	logger := zlog.FromContext(ctx)

	raw, err := req.Body()
	if err != nil {
		logger.Warnf("read body: %v", err)
		return zapi.BasicResponse(http.StatusBadRequest, jsonHeader(), []byte(`{"error":"bad body"}`))
	}
	var pb postBody
	if err := json.Unmarshal(raw, &pb); err != nil || pb.Name == "" {
		return zapi.BasicResponse(http.StatusBadRequest, jsonHeader(), []byte(`{"error":"name required"}`))
	}

	now := time.Now().UTC()
	item := &models.Item{
		ID:       uuid.NewString(),
		Created:  now,
		Name:     pb.Name,
		Deleted:  false,
		Modified: nil,
	}
	if err := zorm.Put(ctx, r.repo, []*models.Item{item}, zorm.PutOptions{}); err != nil {
		logger.Errorf("put item: %v", err)
		return zapi.Response500InternalServerError()
	}
	body, _ := json.Marshal(resources.ItemFromModel(item))
	return zapi.BasicResponse(http.StatusCreated, jsonHeader(), body)
}
