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

			// If specific fields were requested, only copy those fields
			// Otherwise, replace the entire object
			if len(findOpts.Include.Fields) > 0 {
				copyFields(origObj, findVal, findOpts.Include.Fields)
			} else {
				origObj.Elem().Set(findVal.Elem())
			}
		}
	}

	if totalFound != targetVal.Len() {
		return fmt.Errorf("expected %d rows found, but only got %d: %w", targetVal.Len(), totalFound, zorm.ErrNotFound)
	}

	return nil
}

// put performs an upsert operation on the provided models.
//
// # Cascading Relations
//
// When opts.Include.Relations specifies relations, those related models will also
// be Put in the correct order based on foreign key dependencies. FK values are
// automatically copied between parent and related models after each insert.
//
// Relations are navigational - the same FK constraint can be defined from both sides:
//   - Account.Users: accounts.id = users.account_id (navigate from Account to Users)
//   - User.Account: users.account_id = accounts.id (navigate from User to Account)
//
// The Relation.Columns map defines: localColumn → remoteColumn for the join.
//
// # FK Location Detection
//
// To determine which table holds the FK, check if local columns are this model's PK:
//   - If local columns ARE this model's PK → FK is on related model
//   - If remote columns ARE related model's PK → FK is on this model
//
// # Put Ordering
//
// Pattern A: FK is on this model (fkIsLocal=true)
//
//	Example: User.Account where users.account_id → accounts.id
//	Put Account first, copy Account.ID to User.AccountID, then Put User
//
// Pattern B: FK is on related model (fkIsLocal=false)
//
//	Example: Account.Users where accounts.id ← users.account_id
//	Put Account first, copy Account.ID to each User.AccountID, then Put Users
//
// # Limitations
//
// Cascading Put does NOT support scenarios requiring two writes to a single record:
//   - Circular dependencies (A.b_id → B, B.a_id → A) where both FKs are required
//   - Nullable FK workarounds (insert with NULL, insert related, update FK)
//
// These patterns will return an error. Handle them with separate Put calls.
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

	// Categorize relations by FK location for ordering
	beforeRelations, afterRelations, err := mapping.categorizeRelationsForPut(opts.Include.Relations)
	if err != nil {
		return fmt.Errorf("categorizing relations: %w", err)
	}

	for i := 0; i < targetVal.Len(); i++ {
		val := targetVal.Index(i)

		// Step 1: Put "before" relations (FK on this model - related must exist first)
		for _, rel := range beforeRelations {
			if err := r.putRelatedModels(ctx, val, rel, primaryKeyFields); err != nil {
				return fmt.Errorf("putting before-relation %s: %w", rel.fieldName, err)
			}
		}

		// Step 2: Put this model
		if err := r.putSingleModel(ctx, mapping, primaryKeyFields, val, opts.Include.Fields); err != nil {
			return err
		}

		// Step 3: Put "after" relations (FK on related model - this model must exist first)
		for _, rel := range afterRelations {
			if err := r.putRelatedModels(ctx, val, rel, primaryKeyFields); err != nil {
				return fmt.Errorf("putting after-relation %s: %w", rel.fieldName, err)
			}
		}
	}

	// Default GetOptions.Include to PutOptions.Include if not specified
	getOpts := opts.GetOptions
	if getOpts.Include.IsEmpty() {
		getOpts.Include = opts.Include
	}

	err = r.Get(ctx, listOfPtrs, getOpts)
	if err != nil {
		return fmt.Errorf("error in get after put: %w", err)
	}

	return nil
}

// putSingleModel performs the upsert logic for a single model instance.
func (r *queryer) putSingleModel(ctx context.Context, mapping Mapping, primaryKeyFields []string, val reflect.Value, fields zorm.Fields) error {
	// Find the first populated lookup key (PK first, then unique keys)
	key, hasKey, err := mapping.findPopulatedLookupKey(val)
	if err != nil {
		return fmt.Errorf("finding lookup key: %w", err)
	}

	if hasKey {
		// Try to update using the identified key
		affected, err := r.update(ctx, mapping, key.FieldNames, val, fields)
		if err != nil {
			return fmt.Errorf("performing update: %w", err)
		}

		if affected == 0 {
			// Update changed no rows - try insert if key columns allow it
			if !key.CanInsert {
				return fmt.Errorf("no rows affected for update and key columns are not insertable: %w", zorm.ErrNotFound)
			}

			err := r.insert(ctx, mapping, primaryKeyFields, val, fields)
			if err != nil {
				if r.conn.Driver().IsConflictError(err) {
					err = zorm.ErrConflict
				}
				return fmt.Errorf("performing insert after zero-row update: %w", err)
			}
		}
	} else {
		// No key populated - perform insert
		err := r.insert(ctx, mapping, primaryKeyFields, val, fields)
		if err != nil {
			if r.conn.Driver().IsConflictError(err) {
				err = zorm.ErrConflict
			}
			return fmt.Errorf("performing insert: %w", err)
		}
	}

	return nil
}

// putRelatedModels puts the related models for a given relation field.
func (r *queryer) putRelatedModels(ctx context.Context, parentVal reflect.Value, rel relInfo, parentPKFields []string) error {
	fieldVal := parentVal.Elem().FieldByName(rel.fieldName)

	relatedPKFields, err := rel.relatedMapping.primaryKeyFields()
	if err != nil {
		return fmt.Errorf("getting related PK fields: %w", err)
	}

	// Handle both to-one (pointer) and to-many (slice) relations
	var relatedModels []reflect.Value
	isToMany := false
	if fieldVal.Kind() == reflect.Ptr {
		if fieldVal.IsValid() && !fieldVal.IsNil() {
			relatedModels = []reflect.Value{fieldVal}
		}
	} else if fieldVal.Kind() == reflect.Slice {
		isToMany = true
		if fieldVal.IsValid() && !fieldVal.IsNil() {
			for j := 0; j < fieldVal.Len(); j++ {
				relatedModels = append(relatedModels, fieldVal.Index(j))
			}
		}
	}

	// For to-many relations, handle orphan deletion
	if isToMany {
		if err := r.deleteOrphanedRelatedModels(ctx, parentVal, rel, relatedPKFields, relatedModels); err != nil {
			return fmt.Errorf("deleting orphaned related models: %w", err)
		}
	}

	if len(relatedModels) == 0 {
		return nil
	}

	// Copy FK values between parent and related models based on FK location
	for _, relatedVal := range relatedModels {
		if err := rel.copyFKValues(parentVal, relatedVal); err != nil {
			return fmt.Errorf("copying FK values: %w", err)
		}

		// Put the related model
		if err := r.putSingleModel(ctx, rel.relatedMapping, relatedPKFields, relatedVal, rel.includeOpts.Include.Fields); err != nil {
			return fmt.Errorf("putting related model: %w", err)
		}

		// After putting, copy back any generated values (e.g., auto-increment PKs)
		if err := rel.copyFKValues(parentVal, relatedVal); err != nil {
			return fmt.Errorf("copying FK values after put: %w", err)
		}
	}

	return nil
}

// deleteOrphanedRelatedModels deletes related models that exist in the database
// but are not in the provided list. For to-many relations, this ensures the
// database matches exactly what's provided (respecting any Where filter).
func (r *queryer) deleteOrphanedRelatedModels(
	ctx context.Context,
	parentVal reflect.Value,
	rel relInfo,
	relatedPKFields []string,
	providedModels []reflect.Value,
) error {
	// Build the FK constraint: find all related rows for this parent
	// For to-many relations, fkIsLocal is false, so FK is on the related model
	// rel.relation.Columns maps parent column -> related column (e.g., "id" -> "user_id")
	fkWhereClause, err := r.buildFKWhereClause(parentVal, rel)
	if err != nil {
		return fmt.Errorf("building FK where clause: %w", err)
	}

	// Combine FK constraint with optional relation Where filter
	var combinedWhere zclause.Clause
	if rel.includeOpts.Where != nil {
		combinedWhere = zelem.And(fkWhereClause, rel.includeOpts.Where)
	} else {
		combinedWhere = fkWhereClause
	}

	// Find existing related models matching the combined WHERE
	existingModels, err := r.findRelatedModels(ctx, rel.relatedMapping, combinedWhere)
	if err != nil {
		return fmt.Errorf("finding existing related models: %w", err)
	}

	if len(existingModels) == 0 {
		return nil // Nothing to delete
	}

	// Build set of provided PKs (after setting FK values so they're populated)
	providedPKs := make(map[string]bool)
	for _, model := range providedModels {
		// Copy FK values to ensure the model has the parent's FK
		if err := rel.copyFKValues(parentVal, model); err != nil {
			return fmt.Errorf("copying FK values for PK extraction: %w", err)
		}
		pk := extractPKString(model, relatedPKFields)
		if pk != "" {
			providedPKs[pk] = true
		}
	}

	// Find orphans: existing models not in provided set
	var orphans []reflect.Value
	for _, existing := range existingModels {
		pk := extractPKString(existing, relatedPKFields)
		if pk != "" && !providedPKs[pk] {
			orphans = append(orphans, existing)
		}
	}

	// Delete orphans
	if _, err := r.deleteByPK(ctx, rel.relatedMapping, relatedPKFields, orphans); err != nil {
		return fmt.Errorf("deleting orphaned models: %w", err)
	}

	return nil
}

// buildFKWhereClause builds a WHERE clause for the FK constraint.
func (r *queryer) buildFKWhereClause(parentVal reflect.Value, rel relInfo) (zclause.Clause, error) {
	// For to-many relations (fkIsLocal=false), the FK is on the related model
	// rel.relation.Columns maps parent column name -> related column name
	// We need to find the related field name for the FK column
	for localCol, remoteCol := range rel.relation.Columns {
		// Get the parent field value (the PK)
		localFields, err := rel.parentMapping.columnNamesToFields([]string{localCol})
		if err != nil {
			return nil, fmt.Errorf("mapping local column %s: %w", localCol, err)
		}
		parentPKValue := parentVal.Elem().FieldByName(localFields[0]).Interface()

		// Get the related field name for the FK
		remoteFields, err := rel.relatedMapping.columnNamesToFields([]string{remoteCol})
		if err != nil {
			return nil, fmt.Errorf("mapping remote column %s: %w", remoteCol, err)
		}

		// Build WHERE clause: relatedFK = parentPK
		return zelem.Eq(zelem.Field(remoteFields[0]), zelem.Value(parentPKValue)), nil
	}

	return nil, fmt.Errorf("no FK columns found in relation")
}

// findRelatedModels finds models of the given type matching the WHERE clause.
func (r *queryer) findRelatedModels(ctx context.Context, mapping Mapping, where zclause.Clause) ([]reflect.Value, error) {
	// Create a slice to hold results
	sliceType := reflect.SliceOf(reflect.TypeOf(mapping.PtrType))
	resultSlice := reflect.MakeSlice(sliceType, 0, 10)
	resultPtr := reflect.New(sliceType)
	resultPtr.Elem().Set(resultSlice)

	// Find using the mapping's model type
	err := r.find(ctx, resultPtr.Interface(), zorm.FindOptions{
		Where: where,
	})
	if err != nil {
		return nil, fmt.Errorf("finding related models: %w", err)
	}

	// Convert result to []reflect.Value
	resultSlice = resultPtr.Elem()
	result := make([]reflect.Value, resultSlice.Len())
	for i := 0; i < resultSlice.Len(); i++ {
		result[i] = resultSlice.Index(i)
	}

	return result, nil
}

// extractPKString extracts a string representation of the PK for comparison.
func extractPKString(model reflect.Value, pkFields []string) string {
	if model.Kind() == reflect.Ptr {
		if model.IsNil() {
			return ""
		}
		model = model.Elem()
	}

	var parts []string
	for _, field := range pkFields {
		val := model.FieldByName(field)
		if !val.IsValid() || val.IsZero() {
			return "" // PK not set, can't identify
		}
		parts = append(parts, fmt.Sprintf("%v", val.Interface()))
	}
	return strings.Join(parts, "|")
}

// deleteByPK deletes models by their primary keys. Returns the number of rows deleted.
func (r *queryer) deleteByPK(ctx context.Context, mapping Mapping, pkFields []string, models []reflect.Value) (int, error) {
	if len(models) == 0 {
		return 0, nil
	}

	driver := r.conn.Driver()
	targetTable := table{name: mapping.Table}

	// Build PK column names
	pkStructure, err := mapping.mapStructure(targetTable, "", pkFields, zorm.Relations{})
	if err != nil {
		return 0, fmt.Errorf("mapping PK columns: %w", err)
	}

	whereCols := make([]string, 0, len(pkStructure.columns))
	for _, col := range pkStructure.columns {
		whereCols = append(whereCols, col.escaped(driver))
	}

	// Collect all PK values
	values := make([]interface{}, 0, len(models)*len(pkFields))
	for _, model := range models {
		if model.Kind() == reflect.Ptr {
			model = model.Elem()
		}
		for _, field := range pkFields {
			values = append(values, model.FieldByName(field).Interface())
		}
	}

	// Build DELETE query
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE (%s) IN (%s)",
		targetTable.escaped(driver),
		strings.Join(whereCols, ","),
		strings.Join(
			zfunc.MakeSlice(
				"("+strings.Join(zfunc.MakeSlice("?", len(pkFields)), ",")+")",
				len(models),
			),
			",",
		),
	)

	count, _, err := zsql.Exec(ctx, r.conn, query, values)
	if err != nil {
		return 0, fmt.Errorf("executing delete: %w", err)
	}

	return count, nil
}

func (r *queryer) delete(ctx context.Context, listOfPtrs any, opts zorm.DeleteOptions) error {
	targetVal, modelPtrType, err := validateListOfPtr(listOfPtrs)
	if err != nil {
		return fmt.Errorf("invalid argument to delete: %w", err)
	}

	if targetVal.Len() == 0 {
		return nil
	}

	typeID := zreflect.TypeID(modelPtrType)
	mapping, ok := r.cfg.mappings[typeID]
	if !ok {
		return fmt.Errorf("mapping unavailable for type %s", typeID)
	}

	primaryKeyFields, err := mapping.primaryKeyFields()
	if err != nil {
		return fmt.Errorf("mapping primary key for delete: %w", err)
	}

	err = r.Get(ctx, listOfPtrs, opts.GetOptions)
	if err != nil {
		return fmt.Errorf("error in get before delete: %w", err)
	}

	// Convert to []reflect.Value for deleteByPK
	models := make([]reflect.Value, targetVal.Len())
	for i := 0; i < targetVal.Len(); i++ {
		models[i] = targetVal.Index(i)
	}

	count, err := r.deleteByPK(ctx, mapping, primaryKeyFields, models)
	if err != nil {
		return fmt.Errorf("deleting models: %w", err)
	}

	if count != targetVal.Len() {
		return fmt.Errorf("expected %d rows affected, but only got %d: %w", targetVal.Len(), count, zorm.ErrNotFound)
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
