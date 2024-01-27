package zormsql

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"

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
	conn     zsql.Transactor
	mappings map[string]Mapping
}

func NewRepository(name string, conn zsql.Transactor) *Repository {
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

func (r *Repository) Find(ctx context.Context, ptrToListOfPtrs any, opts zorm.FindOptions) (err error) {
	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(error); ok {
				err = fmt.Errorf("panic in find: %w - %s", er, string(debug.Stack()))
			} else {
				err = fmt.Errorf("panic in find: %v - %s", e, string(debug.Stack()))
			}
		}
	}()

	return r.find(ctx, ptrToListOfPtrs, opts)
}

func (r *Repository) Get(ctx context.Context, listOfPtrs any, opts zorm.GetOptions) (err error) {
	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(error); ok {
				err = fmt.Errorf("panic in find: %w - %s", er, string(debug.Stack()))
			} else {
				err = fmt.Errorf("panic in find: %v - %s", e, string(debug.Stack()))
			}
		}
	}()

	return r.get(ctx, listOfPtrs, opts)
}

func (r *Repository) find(ctx context.Context, ptrToListOfPtrs any, opts zorm.FindOptions) error {
	targetList, modelPtrType, err := validatePtrToListOfPtr(ptrToListOfPtrs)
	if err != nil {
		return fmt.Errorf("invalid argument to find: %w", err)
	}

	typeID := zreflect.TypeID(modelPtrType)
	mapping, ok := r.mappings[typeID]
	if !ok {
		return fmt.Errorf("find mapping unavailable type %s", typeID)
	}

	plan, err := buildSelectQueryPlan(r, mapping, opts.Include.Fields, opts.Where, opts.Include.Sort)
	if err != nil {
		return fmt.Errorf("building query plan for find: %w", err)
	}

	limit := fmt.Sprintf(" LIMIT %d OFFSET %d", targetList.Cap(), opts.Offset)

	query := plan.query(limit)

	// fmt.Printf("Q: %s\nV: %s", query, plan.values)

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

func (r *Repository) get(ctx context.Context, listOfPtrs any, opts zorm.GetOptions) error {
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
		objPtr := targetVal.Index(i)

		fieldValues := extractFields(pk, objPtr)
		pkID, err := json.Marshal(fieldValues)
		if err != nil {
			return fmt.Errorf("rendering primary key values into string key for get query: %w", err)
		}

		values := zfunc.Map(fieldValues, func(v any) zelement.Element { return zelem.Value(v) })
		pkValues = append(pkValues, values)

		objMap.SetMapIndex(reflect.ValueOf(string(pkID)), objPtr)
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

		pkKey := reflect.ValueOf(string(pkID))
		mapValue := objMap.MapIndex(pkKey)
		if !mapValue.IsValid() {
			return fmt.Errorf("cannot process get results in find, found unexpected model identified by primary key: %s", pkID)
		}
		mapValue.Elem().Set(findVal.Elem())
	}

	return nil
}

func (r *Repository) Put(ctx context.Context, listOfPtrs any, opts zorm.PutOptions) error {
	targetVal, modelPtrType, err := validateListOfPtr(listOfPtrs)
	if err != nil {
		return fmt.Errorf("invalid argument to get: %w", err)
	}

	typeID := zreflect.TypeID(modelPtrType)
	mapping, ok := r.mappings[typeID]
	if !ok {
		return fmt.Errorf("get mapping unavailable for type %s", typeID)
	}

	primaryKeyFields, err := mapping.primaryKeyFields()
	if err != nil {
		return fmt.Errorf("mapping primary key for get in clause: %w", err)
	}

	for i := 0; i < targetVal.Len(); i++ {
		val := targetVal.Index(i)
		doUpdate := mapping.hasValues(val, primaryKeyFields)
		if doUpdate {
			err := r.update(ctx, mapping, primaryKeyFields, val)
			if err != nil {
				return fmt.Errorf("performing update: %w", err)
			}
		} else {
			err := r.insert(ctx, mapping, primaryKeyFields, val)
			if err != nil {
				return fmt.Errorf("performing insert: %w", err)
			}
		}
	}

	err = r.Get(ctx, listOfPtrs, opts.GetOptions)
	if err != nil {
		return fmt.Errorf("error in get after put: %w", err)
	}

	return nil
}

func (r *Repository) insert(ctx context.Context, mapping Mapping, primaryKeyFields []string, objPtr reflect.Value) error {
	fields := mapping.insertFields()
	columns, _, err := mapping.mapFields(r.conn.Driver(), "", "", fields)
	if err != nil {
		return fmt.Errorf("mapping insert columns: %w", err)
	}

	query := fmt.Sprintf(
		`
		INSERT INTO
		%s
		(%s)
		VALUES
		(%s)
		`,
		r.conn.Driver().EscapeTable(mapping.Table),
		strings.Join(columns, ","),
		strings.Join(zfunc.MakeSlice("?", len(columns)), ","),
	)

	values := make([]interface{}, 0, len(fields))
	for _, f := range fields {
		values = append(values, objPtr.Elem().FieldByName(f).Interface())
	}

	fmt.Printf("%s\n%v\n", query, values)
	_, id, err := zsql.Exec(ctx, r.conn, query, values)
	if err != nil {
		return fmt.Errorf("executing insert: %w", err)
	}

	if len(primaryKeyFields) == 1 && id != 0 {
		field := objPtr.Elem().FieldByName(primaryKeyFields[0])

		if zreflect.IsInt(field.Type()) {
			idVal := reflect.ValueOf(&id).Elem()
			field.Set(idVal)
		} else if zreflect.IsString(field.Type()) {
			idStrVal := fmt.Sprintf("%d", id)
			idVal := reflect.ValueOf(&idStrVal).Elem()
			field.Set(idVal)
		} else {
			return fmt.Errorf("cannot populate primary key field that isn't int or string type: %v", field.Type())
		}
	}

	return nil
}

func (r *Repository) update(ctx context.Context, mapping Mapping, primaryKeyFields []string, objPtr reflect.Value) error {
	primaryKeyColumns, _, err := mapping.mapFields(r.conn.Driver(), "", "", primaryKeyFields)
	if err != nil {
		return fmt.Errorf("mapping primary key columns for update: %w", err)
	}

	fields := mapping.updateFields()
	columns, _, err := mapping.mapFields(r.conn.Driver(), "", "", fields)
	if err != nil {
		return fmt.Errorf("mapping update columns: %w", err)
	}

	query := fmt.Sprintf(
		`
		UPDATE
		%s
		SET
		%s
		WHERE
		%s
		`,
		r.conn.Driver().EscapeTable(mapping.Table),
		strings.Join(zfunc.Map(columns, func(c string) string {
			return fmt.Sprintf(
				"%s=?",
				c,
			)
		}), ", "),
		strings.Join(zfunc.Map(primaryKeyColumns, func(c string) string {
			return fmt.Sprintf(
				"%s %s ?",
				c,
				r.conn.Driver().NullSafeEqualityOperator(),
			)
		}), " AND "),
	)

	values := make([]any, 0, len(fields)+len(primaryKeyFields))
	for _, f := range append(fields, primaryKeyFields...) {
		values = append(values, objPtr.Elem().FieldByName(f).Interface())
	}

	fmt.Printf("%s\n%v\n", query, values)
	affected, _, err := zsql.Exec(ctx, r.conn, query, values)
	if err != nil {
		return fmt.Errorf("executing insert: %w", err)
	}

	if affected == 0 {
		return zorm.ErrNotFound
	}

	if affected != 1 {
		return fmt.Errorf("more than one row (%d) affected by model update query (!!!)", affected)
	}

	return nil
}
