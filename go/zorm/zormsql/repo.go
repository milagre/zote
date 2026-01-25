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

type Config struct {
	name     string
	mappings map[string]Mapping
}

// Repository
type Repository struct {
	*queryer
	ts zsql.Transactor
}

type Transaction struct {
	*queryer
	tx zsql.Transaction
}

type queryer struct {
	cfg  *Config
	conn zsql.QueryExecutor
}

func NewRepository(name string, conn zsql.Transactor) *Repository {
	cfg := &Config{
		name:     name,
		mappings: map[string]Mapping{},
	}
	return &Repository{
		queryer: &queryer{
			cfg:  cfg,
			conn: conn,
		},
		ts: conn,
	}
}

func (r *Repository) AddMapping(m Mapping) {
	if r.cfg.mappings == nil {
		r.cfg.mappings = map[string]Mapping{}
	}

	key := zreflect.TypeID(reflect.TypeOf(m.PtrType))
	if _, ok := r.cfg.mappings[key]; ok {
		panic(fmt.Sprintf("Duplicate sql mapping for type %s", key))
	}

	m.repo = r
	r.cfg.mappings[key] = m
}

func (r *Repository) Begin(ctx context.Context) (zorm.Transaction, error) {
	tx, err := r.ts.Begin(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("startin transaction: %w", err)
	}

	return &Transaction{
		queryer: &queryer{
			cfg:  r.cfg,
			conn: tx,
		},
		tx: tx,
	}, nil
}

func (t *Transaction) Commit() error {
	err := t.tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

func (t *Transaction) Rollback() error {
	err := t.tx.Rollback()
	if err != nil {
		return fmt.Errorf("rolling back transaction: %w", err)
	}

	return nil
}

func (r *queryer) Find(ctx context.Context, ptrToListOfPtrs any, opts zorm.FindOptions) (err error) {
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

func (r *queryer) Get(ctx context.Context, listOfPtrs any, opts zorm.GetOptions) (err error) {
	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(error); ok {
				err = fmt.Errorf("panic in get: %w - %s", er, string(debug.Stack()))
			} else {
				err = fmt.Errorf("panic in get: %v - %s", e, string(debug.Stack()))
			}
		}
	}()

	return r.get(ctx, listOfPtrs, opts)
}

func (r *queryer) Put(ctx context.Context, listOfPtrs any, opts zorm.PutOptions) (err error) {
	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(error); ok {
				err = fmt.Errorf("panic in put: %w - %s", er, string(debug.Stack()))
			} else {
				err = fmt.Errorf("panic in put: %v - %s", e, string(debug.Stack()))
			}
		}
	}()

	return r.put(ctx, listOfPtrs, opts)
}

func (r *queryer) Delete(ctx context.Context, listOfPtrs any, opts zorm.DeleteOptions) (err error) {
	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(error); ok {
				err = fmt.Errorf("panic in delete: %w - %s", er, string(debug.Stack()))
			} else {
				err = fmt.Errorf("panic in delete: %v - %s", e, string(debug.Stack()))
			}
		}
	}()

	return r.delete(ctx, listOfPtrs, opts)
}

func (r *queryer) find(ctx context.Context, ptrToListOfPtrs any, opts zorm.FindOptions) error {
	targetList, modelPtrType, err := validatePtrToListOfPtr(ptrToListOfPtrs)
	if err != nil {
		return fmt.Errorf("invalid argument to find: %w", err)
	}

	typeID := zreflect.TypeID(modelPtrType)
	mapping, ok := r.cfg.mappings[typeID]
	if !ok {
		return fmt.Errorf("find mapping unavailable type %s", typeID)
	}

	plan, err := buildSelectQueryPlan(r, mapping, opts.Include.Fields, opts.Include.Relations, opts.Where, opts.Sort, targetList.Cap(), opts.Offset)
	if err != nil {
		return fmt.Errorf("building query plan for find: %w", err)
	}

	query, values := plan.query(r.conn.Driver())

	rows, err := r.conn.Query(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("executing find query: %w", err)
	}
	defer rows.Close()

	err = plan.process(targetList, rows)
	if err != nil {
		return fmt.Errorf("find read error: %w", err)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("find rows error: %w", err)
	}

	return nil
}

func (r *queryer) get(ctx context.Context, listOfPtrs any, opts zorm.GetOptions) error {
	targetVal, modelPtrType, err := validateListOfPtr(listOfPtrs)
	if err != nil {
		return fmt.Errorf("invalid argument to get: %w", err)
	}

	if reflect.ValueOf(listOfPtrs).Len() == 0 {
		return nil
	}

	typeID := zreflect.TypeID(modelPtrType)
	mapping, ok := r.cfg.mappings[typeID]
	if !ok {
		return fmt.Errorf("mapping unavailable for type %s", typeID)
	}

	// Group objects by their populated lookup key
	type keyGroup struct {
		keyFields []string
		keys      []lookupKey
		objMap    map[string]reflect.Value // maps ValuesKey to original object pointers
	}
	groups := make(map[string]*keyGroup)

	for i := 0; i < targetVal.Len(); i++ {
		objPtr := targetVal.Index(i)

		key, hasKey, err := mapping.findPopulatedLookupKey(objPtr)
		if err != nil {
			return fmt.Errorf("finding lookup key for get: %w", err)
		}

		if !hasKey {
			return fmt.Errorf("no populated key found for object at index %d", i)
		}

		group, ok := groups[key.GroupKey]
		if !ok {
			group = &keyGroup{
				keyFields: key.FieldNames,
				keys:      make([]lookupKey, 0),
				objMap:    make(map[string]reflect.Value),
			}
			groups[key.GroupKey] = group
		}

		group.keys = append(group.keys, key)
		group.objMap[key.ValuesKey] = objPtr
	}

	totalFound := 0

	// Query each group separately
	for _, group := range groups {
		keyFields := group.keyFields

		// Build IN clause values from pre-computed key values
		keyValues := make([][]zelement.Element, 0, len(group.keys))
		for _, key := range group.keys {
			values := zfunc.Map(key.Values, func(v any) zelement.Element { return zelem.Value(v) })
			keyValues = append(keyValues, values)
		}

		where := zclause.In{
			Left:  zfunc.Map(keyFields, func(f string) zelement.Element { return zelem.Field(f) }),
			Right: keyValues,
		}

		findOpts := zorm.FindOptions{
			Include: opts.Include,
			Where:   where,
		}
		if len(findOpts.Include.Fields) > 0 {
			findOpts.Include.Fields.Add(keyFields...)
		}

		findTarget := zreflect.MakeAddressableSliceOf(modelPtrType, 0, len(group.keys))

		err = r.Find(ctx, findTarget.Addr().Interface(), findOpts)
		if err != nil {
			return fmt.Errorf("executing find for get: %w", err)
		}

		totalFound += findTarget.Len()

		// Map results back to original objects
		for i := 0; i < findTarget.Len(); i++ {
			findVal := findTarget.Index(i)
			fieldValues := extractFields(keyFields, findVal)
			mapKey, err := json.Marshal(fieldValues)
			if err != nil {
				return fmt.Errorf("rendering key values into string key for get results: %w", err)
			}

			origObj, ok := group.objMap[string(mapKey)]
			if !ok {
				return fmt.Errorf("cannot process get results in find, found unexpected model identified by key: %s", mapKey)
			}
			origObj.Elem().Set(findVal.Elem())
		}
	}

	if totalFound != targetVal.Len() {
		return fmt.Errorf("expected %d rows found, but only got %d: %w", targetVal.Len(), totalFound, zorm.ErrNotFound)
	}

	return nil
}

func (r *queryer) put(ctx context.Context, listOfPtrs any, opts zorm.PutOptions) error {
	targetVal, modelPtrType, err := validateListOfPtr(listOfPtrs)
	if err != nil {
		return fmt.Errorf("invalid argument to put: %w", err)
	}

	if reflect.ValueOf(listOfPtrs).Len() == 0 {
		return nil
	}

	typeID := zreflect.TypeID(modelPtrType)
	mapping, ok := r.cfg.mappings[typeID]
	if !ok {
		return fmt.Errorf("mapping unavailable for type %s", typeID)
	}

	primaryKeyFields, err := mapping.primaryKeyFields()
	if err != nil {
		return fmt.Errorf("mapping primary key for put: %w", err)
	}

	for i := 0; i < targetVal.Len(); i++ {
		val := targetVal.Index(i)

		// Find the first populated lookup key (PK first, then unique keys)
		key, hasKey, err := mapping.findPopulatedLookupKey(val)
		if err != nil {
			return fmt.Errorf("finding lookup key: %w", err)
		}

		if hasKey {
			// Try to update using the identified key
			affected, err := r.update(ctx, mapping, key.FieldNames, val, opts.Include.Fields)
			if err != nil {
				return fmt.Errorf("performing update: %w", err)
			}

			if affected == 0 {
				// Update changed no rows - try insert if key columns allow it
				if !key.CanInsert {
					return fmt.Errorf("no rows affected for update and key columns are not insertable: %w", zorm.ErrNotFound)
				}

				err := r.insert(ctx, mapping, primaryKeyFields, val, opts.Include.Fields)
				if err != nil {
					if r.conn.Driver().IsConflictError(err) {
						err = zorm.ErrConflict
					}
					return fmt.Errorf("performing insert after zero-row update: %w", err)
				}
			}
		} else {
			// No key populated - perform insert
			err := r.insert(ctx, mapping, primaryKeyFields, val, opts.Include.Fields)
			if err != nil {
				if r.conn.Driver().IsConflictError(err) {
					err = zorm.ErrConflict
				}
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

func (r *queryer) delete(ctx context.Context, listOfPtrs any, opts zorm.DeleteOptions) error {
	targetVal, modelPtrType, err := validateListOfPtr(listOfPtrs)
	if err != nil {
		return fmt.Errorf("invalid argument to delete: %w", err)
	}

	if reflect.ValueOf(listOfPtrs).Len() == 0 {
		return nil
	}

	typeID := zreflect.TypeID(modelPtrType)
	mapping, ok := r.cfg.mappings[typeID]
	if !ok {
		return fmt.Errorf("mapping unavailable for type %s", typeID)
	}

	driver := r.conn.Driver()

	targetTable := table{
		name: mapping.Table,
	}

	primaryKeyFields, err := mapping.primaryKeyFields()
	if err != nil {
		return fmt.Errorf("mapping primary key for delete in clause: %w", err)
	}

	primaryKeyStructure, err := mapping.mapStructure(targetTable, "", primaryKeyFields, zorm.Relations{})
	if err != nil {
		return fmt.Errorf("mapping primary key columns for delete: %w", err)
	}

	err = r.Get(ctx, listOfPtrs, opts.GetOptions)
	if err != nil {
		return fmt.Errorf("error in get before delete: %w", err)
	}

	values := make([]interface{}, 0, targetVal.Len()*len(primaryKeyFields))
	for i := 0; i < targetVal.Len(); i++ {
		val := targetVal.Index(i)

		primaryKeyValue := make([]interface{}, 0, len(primaryKeyFields))
		for _, f := range primaryKeyFields {
			primaryKeyValue = append(primaryKeyValue, val.Elem().FieldByName(f).Interface())
		}

		values = append(values, primaryKeyValue...)
	}

	whereCols := make([]string, 0, len(primaryKeyStructure.columns))
	for _, col := range primaryKeyStructure.columns {
		whereCols = append(whereCols, col.escaped(driver))
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE (%s) IN (%s)",
		targetTable.escaped(driver),
		strings.Join(whereCols, ","),
		strings.Join(
			zfunc.MakeSlice(
				"("+strings.Join(zfunc.MakeSlice("?", len(primaryKeyFields)), ",")+")",
				targetVal.Len(),
			),
			",",
		),
	)

	// fmt.Printf("Q: %s\nV: %s", query, values)

	count, _, err := zsql.Exec(ctx, r.conn, query, values)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	if count != targetVal.Len() {
		return fmt.Errorf("expected %d rows affected, but only got %d: %w", count, targetVal.Len(), zorm.ErrNotFound)
	}

	return nil
}

func (r *queryer) insert(ctx context.Context, mapping Mapping, primaryKeyFields []string, objPtr reflect.Value, fields zorm.Fields) error {
	targetTable := table{
		name: mapping.Table,
	}

	fields, columns := mapping.insertFields(fields)
	queryColumns := make([]string, 0, len(columns))
	for _, col := range columns {
		queryColumns = append(queryColumns, col.escaped(r.conn.Driver()))
	}

	query := fmt.Sprintf(
		`
		INSERT INTO
		%s
		(%s)
		VALUES
		(%s)
		`,
		targetTable.escaped(r.conn.Driver()),
		strings.Join(queryColumns, ","),
		strings.Join(zfunc.MakeSlice("?", len(queryColumns)), ","),
	)

	values := make([]interface{}, 0, len(fields))
	for _, f := range fields {
		values = append(values, objPtr.Elem().FieldByName(f).Interface())
	}

	// fmt.Printf("Q: %s\nV: %s", query, plan.values)

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

func (r *queryer) update(ctx context.Context, mapping Mapping, keyFields []string, objPtr reflect.Value, fields zorm.Fields) (int, error) {
	driver := r.conn.Driver()
	targetTable := table{
		name: mapping.Table,
	}

	fields = mapping.updateFields(fields)
	structure, err := mapping.mapStructure(targetTable, "", fields, zorm.Relations{})
	if err != nil {
		return 0, fmt.Errorf("mapping update columns: %w", err)
	}

	// Map key fields to columns
	keyColumns := make([]column, 0, len(keyFields))
	for _, kf := range keyFields {
		for _, c := range mapping.Columns {
			if c.Field == kf {
				keyColumns = append(keyColumns, column{
					table: targetTable,
					name:  c.Name,
				})
				break
			}
		}
	}

	if len(keyColumns) != len(keyFields) {
		return 0, fmt.Errorf("could not map all key fields to columns")
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
		targetTable.escaped(driver),
		strings.Join(zfunc.Map(structure.columns, func(c column) string {
			return fmt.Sprintf(
				"%s=?",
				c.escaped(driver),
			)
		}), ", "),
		strings.Join(zfunc.Map(keyColumns, func(c column) string {
			return fmt.Sprintf(
				"%s %s ?",
				c.escaped(driver),
				driver.NullSafeEqualityOperator(),
			)
		}), " AND "),
	)

	values := make([]any, 0, len(fields)+len(keyFields))
	for _, f := range append(fields, keyFields...) {
		values = append(values, objPtr.Elem().FieldByName(f).Interface())
	}

	affected, _, err := zsql.Exec(ctx, r.conn, query, values)
	if err != nil {
		return 0, fmt.Errorf("executing update: %w", err)
	}

	if affected > 1 {
		return affected, fmt.Errorf("more than one row (%d) affected by model update query (!!!)", affected)
	}

	return affected, nil
}
