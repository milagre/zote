package zormsql

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zfunc"
	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zreflect"
	"github.com/milagre/zote/go/zsql"
)

// Repository
type Repository struct {
	name     string
	conn     zsql.Connection
	mappings map[string]Mapping
}

func NewRepository(name string, conn zsql.Connection) *Repository {
	return &Repository{
		name:     name,
		conn:     conn,
		mappings: map[string]Mapping{},
	}
}

func (r *Repository) AddMapping(m Mapping) {
	if r.mappings == nil {
		r.mappings = map[string]Mapping{}
	}
	r.mappings[zreflect.TypeID(reflect.TypeOf(m.PtrType))] = m
}

func (r *Repository) Find(ctx context.Context, ptrToListOfPtrs any, opts zorm.FindOptions) error {
	targetList, modelPtrType, err := validatePtrToListOfPtr(ptrToListOfPtrs)
	if err != nil {
		return fmt.Errorf("invalid argument to find: %w", err)
	}

	typeID := zreflect.TypeID(modelPtrType)
	mapping, ok := r.mappings[typeID]
	if !ok {
		return fmt.Errorf("find mapping unavailable type %s", typeID)
	}

	plan, err := buildSelectQueryPlan(r, mapping, opts.Include.Fields, opts.Include.Where, opts.Include.Sort)
	if err != nil {
		return fmt.Errorf("building query plan for find: %w", err)
	}

	limit := fmt.Sprintf(" LIMIT %d OFFSET %d", targetList.Cap(), opts.Offset)

	query := plan.query(limit)

	// fmt.Printf("Q: %s\n", query)

	rows, err := r.conn.Query(ctx, query, plan.values...)
	if err != nil {
		return fmt.Errorf("executing find query: %w", err)
	}
	defer rows.Close()

	var obj reflect.Value
	var count int
	var currentPrimaryKey string
	scanTarget := append(plan.primaryKeyTarget, plan.target...)
	for rows.Next() {
		err := rows.Scan(scanTarget...)
		if err != nil {
			return fmt.Errorf("scanning find result row: %w", err)
		}

		isNew := false
		newPrimaryKey, err := plan.scannedPrimaryKey()
		if err != nil {
			return fmt.Errorf("creating find primary slug: %w", err)
		}

		if count == 0 || newPrimaryKey != currentPrimaryKey {
			isNew = true
			obj = reflect.New(modelPtrType.Elem()).Elem()
		}

		plan.load(obj)

		if isNew {
			count++
			targetList.SetLen(count)
			targetList.Index(count - 1).Set(obj.Addr())
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("find rows error: %w", err)
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, listOfPtrs any, opts zorm.GetOptions) error {
	targetVal, modelPtrType, err := validateListOfPtr(listOfPtrs)
	if err != nil {
		return fmt.Errorf("invalid argument to get: %w", err)
	}

	typeID := zreflect.TypeID(modelPtrType)
	mapping, ok := r.mappings[typeID]
	if !ok {
		return fmt.Errorf("get mapping unavailable for type %s", typeID)
	}

	pk, err := mapping.primaryKeyFields()
	if err != nil {
		return fmt.Errorf("mapping primary key for get in clause: %w", err)
	}

	objMap := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), modelPtrType))

	pkValues := make([][]zelement.Element, 0, targetVal.Len())
	for i := 0; i < targetVal.Len(); i++ {
		val := targetVal.Index(i)

		fieldValues := extractFields(pk, val)
		pkID, err := json.Marshal(fieldValues)
		if err != nil {
			return fmt.Errorf("rendering primary key values into string key for get query: %w", err)
		}

		values := zfunc.Map(fieldValues, func(v any) zelement.Element { return zelem.Value(v) })
		pkValues = append(pkValues, values)

		objMap.SetMapIndex(reflect.ValueOf(string(pkID)), val)
	}

	where := zclause.In{
		Left:  zfunc.Map(pk, func(f string) zelement.Element { return zelem.Field(f) }),
		Right: pkValues,
	}

	findOpts := zorm.FindOptions{
		Include: opts.Include,
		Where:   where,
	}
	if len(findOpts.Include.Fields) > 0 {
		findOpts.Include.Fields.Add(pk...)
	}

	findTarget := zreflect.MakeAddressableSliceOf(modelPtrType, 0, targetVal.Len())

	err = r.Find(ctx, findTarget.Addr().Interface(), findOpts)
	if err != nil {
		return fmt.Errorf("executing find for get: %w", err)
	}

	if findTarget.Len() != targetVal.Len() {
		return zorm.ErrNotFound
	}

	for i := 0; i < findTarget.Len(); i++ {
		findVal := findTarget.Index(i)
		fieldValues := extractFields(pk, findVal)
		pkID, err := json.Marshal(fieldValues)
		if err != nil {
			return fmt.Errorf("rendering primary key values into string key for get results: %w", err)
		}

		objMap.MapIndex(reflect.ValueOf(string(pkID))).Elem().Set(findVal.Elem())
	}

	return nil
}
