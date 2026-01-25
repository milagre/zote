package zormsql

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/milagre/zote/go/zfunc"
	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zreflect"
)

// Mapping defines how a Go struct maps to a database table.
//
// Example:
//
//	var UserMapping = zormsql.Mapping{
//	    PtrType:    &User{},
//	    Table:      "users",
//	    PrimaryKey: []string{"id"},
//	    UniqueKeys: [][]string{{"email"}},
//	    Columns: []zormsql.Column{
//	        {Name: "id", Field: "ID", NoInsert: true, NoUpdate: true},
//	        {Name: "email", Field: "Email"},
//	        {Name: "account_id", Field: "AccountID"},
//	    },
//	    Relations: []zormsql.Relation{
//	        {Table: "accounts", Columns: map[string]string{"account_id": "id"}, Field: "Account"},
//	    },
//	}
type Mapping struct {
	// PtrType is a pointer to a zero-value instance of the mapped struct (e.g., &User{}).
	PtrType interface{}

	// Table is the database table name.
	Table string

	// PrimaryKey lists the column names (not field names) that form the primary key.
	PrimaryKey []string

	// UniqueKeys lists additional unique constraints, each as a slice of column names.
	// Used for upsert lookups when the primary key is not populated.
	UniqueKeys [][]string

	// Columns defines the mapping between database columns and struct fields.
	Columns []Column

	// Relations defines navigational relationships to other mapped models.
	Relations []Relation

	repo *Repository
}

// Column defines the mapping between a single database column and a struct field.
type Column struct {
	// Name is the database column name.
	Name string

	// Field is the Go struct field name.
	Field string

	// NoInsert indicates this column should be excluded from INSERT statements.
	// Use for auto-generated columns like auto-increment IDs or database-managed timestamps.
	NoInsert bool

	// NoUpdate indicates this column should be excluded from UPDATE statements.
	// Use for immutable columns like primary keys or creation timestamps.
	NoUpdate bool
}

// Relation defines a navigational relationship from this model to another.
//
// The same underlying FK constraint can be defined from both sides:
//   - Account.Users: accounts.id = users.account_id (one-to-many)
//   - User.Account: users.account_id = accounts.id (many-to-one)
//
// Example (many-to-one, FK on this model):
//
//	Relation{
//	    Table:   "accounts",
//	    Columns: map[string]string{"account_id": "id"},  // users.account_id = accounts.id
//	    Field:   "Account",
//	}
//
// Example (one-to-many, FK on related model):
//
//	Relation{
//	    Table:   "users",
//	    Columns: map[string]string{"id": "account_id"},  // accounts.id = users.account_id
//	    Field:   "Users",
//	}
type Relation struct {
	// Table is the related table name.
	Table string

	// Columns maps this model's column names to the related model's column names.
	// The map key is a column on this model's table; the value is a column on the related table.
	Columns map[string]string

	// Field is the Go struct field name for the relation (pointer for to-one, slice for to-many).
	Field string
}

func (m Mapping) relationByField(f string) (Relation, bool) {
	for _, r := range m.Relations {
		if r.Field == f {
			return r, true
		}
	}
	return Relation{}, false
}

// columnsMatchPrimaryKey checks if the given column names exactly match this mapping's primary key.
func (m Mapping) columnsMatchPrimaryKey(columnNames []string) bool {
	if len(columnNames) != len(m.PrimaryKey) {
		return false
	}
	pkSet := make(map[string]bool, len(m.PrimaryKey))
	for _, pk := range m.PrimaryKey {
		pkSet[pk] = true
	}
	for _, col := range columnNames {
		if !pkSet[col] {
			return false
		}
	}
	return true
}

// foreignKeyIsLocal returns true if the FK column(s) are on this model's table.
//
// Example where FK is local (returns true):
//
//	User.Account where users.account_id → accounts.id
//
// Example where FK is on related model (returns false):
//
//	Account.Users where accounts.id ← users.account_id
func (m Mapping) foreignKeyIsLocal(rel Relation) bool {
	localCols := make([]string, 0, len(rel.Columns))
	for localCol := range rel.Columns {
		localCols = append(localCols, localCol)
	}

	// If local columns match this model's PK, FK is on the related model
	return !m.columnsMatchPrimaryKey(localCols)
}

// categorizeRelationsForPut categorizes relations into "before" and "after" groups
// based on FK location, determining the order in which related models should be put.
//
// Returns:
//   - beforeRelations: FK is on this model - related models must be put first
//   - afterRelations: FK is on related model - related models are put after this model
func (m Mapping) categorizeRelationsForPut(relations zorm.Relations) (beforeRelations, afterRelations []relInfo, err error) {
	for fieldName, relOpts := range relations {
		rel, ok := m.relationByField(fieldName)
		if !ok {
			return nil, nil, fmt.Errorf("relation %s not found in mapping", fieldName)
		}

		// Get the related model's mapping
		structField, ok := reflect.TypeOf(m.PtrType).Elem().FieldByName(fieldName)
		if !ok {
			return nil, nil, fmt.Errorf("field %s not found on struct", fieldName)
		}

		var relatedPtrType reflect.Type
		if structField.Type.Kind() == reflect.Ptr {
			relatedPtrType = structField.Type
		} else if structField.Type.Kind() == reflect.Slice {
			relatedPtrType = structField.Type.Elem()
		} else {
			return nil, nil, fmt.Errorf("relation field %s must be pointer or slice", fieldName)
		}

		relatedMapping, ok := m.repo.cfg.mappings[zreflect.TypeID(relatedPtrType)]
		if !ok {
			return nil, nil, fmt.Errorf("mapping unavailable for related type %s", zreflect.TypeID(relatedPtrType))
		}

		fkIsLocal := m.foreignKeyIsLocal(rel)
		info := relInfo{
			fieldName:      fieldName,
			relation:       rel,
			parentMapping:  m,
			relatedMapping: relatedMapping,
			fkIsLocal:      fkIsLocal,
			includeOpts:    relOpts,
		}

		if fkIsLocal {
			beforeRelations = append(beforeRelations, info)
		} else {
			afterRelations = append(afterRelations, info)
		}
	}

	return beforeRelations, afterRelations, nil
}

func (m Mapping) hasValues(objPtr reflect.Value, fields []string) bool {
	for _, f := range fields {
		if objPtr.Elem().FieldByName(f).IsZero() {
			return false
		}
	}
	return true
}

func (m Mapping) allFields() []string {
	result := make([]string, 0, len(m.Columns))
	for _, c := range m.Columns {
		result = append(result, c.Field)
	}
	return result
}

func (m Mapping) insertFields(requestedFields zorm.Fields) ([]string, []column) {
	fields := make([]string, 0, len(m.Columns))
	columns := make([]column, 0, len(m.Columns))

	// Get all insertable fields
	allInsertableFields := make([]string, 0, len(m.Columns))
	for _, c := range m.Columns {
		if !c.NoInsert {
			allInsertableFields = append(allInsertableFields, c.Field)
		}
	}

	// Resolve requested fields (handles negation)
	resolvedFields := requestedFields.Resolve(allInsertableFields)

	for _, f := range resolvedFields {
		for _, c := range m.Columns {
			if f == c.Field && !c.NoInsert {
				fields = append(fields, c.Field)
				columns = append(columns, column{
					table: table{
						name: m.Table,
					},
					name: c.Name,
				})
			}
		}
	}

	return fields, columns
}

func (m Mapping) updateFields(requestedFields zorm.Fields) []string {
	result := make([]string, 0, len(m.Columns))

	// Get all updatable fields
	allUpdatableFields := make([]string, 0, len(m.Columns))
	for _, c := range m.Columns {
		if !c.NoUpdate {
			allUpdatableFields = append(allUpdatableFields, c.Field)
		}
	}

	// Resolve requested fields (handles negation)
	resolvedFields := requestedFields.Resolve(allUpdatableFields)

	for _, f := range resolvedFields {
		for _, c := range m.Columns {
			if f == c.Field && !c.NoUpdate {
				result = append(result, c.Field)
			}
		}
	}

	return result
}

// columnNamesToFields converts column names to their corresponding struct field names.
func (m Mapping) columnNamesToFields(columnNames []string) ([]string, error) {
	result := make([]string, 0, len(columnNames))
	for _, colName := range columnNames {
		for _, col := range m.Columns {
			if colName == col.Name {
				result = append(result, col.Field)
				break
			}
		}
	}

	if len(result) != len(columnNames) {
		return nil, fmt.Errorf("columns not fully mapped for %T", m.PtrType)
	}

	return result, nil
}

func (m Mapping) primaryKeyFields() ([]string, error) {
	return m.columnNamesToFields(m.PrimaryKey)
}

// keyColumnsCanInsert checks if all columns in a key are insertable (not marked NoInsert).
func (m Mapping) keyColumnsCanInsert(columnNames []string) bool {
	for _, keyCol := range columnNames {
		for _, col := range m.Columns {
			if keyCol == col.Name {
				if col.NoInsert {
					return false
				}
				break
			}
		}
	}
	return true
}

// findPopulatedLookupKey finds the first populated key for a given object.
// It first checks the primary key, then each unique key in order.
// Returns the key info and whether a key was found.
func (m Mapping) findPopulatedLookupKey(objPtr reflect.Value) (lookupKey, bool, error) {
	// First, check primary key
	pkFields, err := m.primaryKeyFields()
	if err != nil {
		return lookupKey{}, false, fmt.Errorf("getting primary key fields: %w", err)
	}

	if m.hasValues(objPtr, pkFields) {
		values := extractFields(pkFields, objPtr)
		key, err := newLookupKey(m.PrimaryKey, pkFields, values, m.keyColumnsCanInsert(m.PrimaryKey))
		if err != nil {
			return lookupKey{}, false, fmt.Errorf("creating primary key lookup: %w", err)
		}
		return key, true, nil
	}

	// Then, check each unique key in order
	for _, ukCols := range m.UniqueKeys {
		ukFields, err := m.columnNamesToFields(ukCols)
		if err != nil {
			return lookupKey{}, false, fmt.Errorf("getting unique key fields: %w", err)
		}

		if m.hasValues(objPtr, ukFields) {
			values := extractFields(ukFields, objPtr)
			key, err := newLookupKey(ukCols, ukFields, values, m.keyColumnsCanInsert(ukCols))
			if err != nil {
				return lookupKey{}, false, fmt.Errorf("creating unique key lookup: %w", err)
			}
			return key, true, nil
		}
	}

	return lookupKey{}, false, nil
}

func (m Mapping) mapField(table table, columnAliasPrefix string, field string) (column, interface{}, error) {
	for _, c := range m.Columns {
		if field == c.Field {
			col := c.Name
			if columnAliasPrefix != "" {
				col = columnAliasPrefix + "_" + col
			}

			structField, ok := reflect.TypeOf(m.PtrType).Elem().FieldByName(field)
			if !ok {
				return column{}, nil, fmt.Errorf("mapping field: getting struct field %s on %T", field, m.PtrType)
			}

			res := column{table: table, name: c.Name, alias: col}

			val := reflect.New(structField.Type).Interface()

			return res, val, nil
		}
	}

	return column{}, nil, fmt.Errorf("field %s is not mapped", field)
}

func (m Mapping) mapStructure(tbl table, columnAliasPrefix string, requestedFields zorm.Fields, relations zorm.Relations) (structure, error) {
	ptrType := reflect.TypeOf(m.PtrType)

	// Resolve requested fields (handles negation)
	fields := requestedFields.Resolve(m.allFields())

	pkf, err := m.primaryKeyFields()
	if err != nil {
		return structure{}, fmt.Errorf("cannot locate primary key fields: %w", err)
	}
	pkLength := len(pkf)
	fields = append(pkf, fields...)

	columns := make([]column, 0, len(fields))
	target := make([]interface{}, 0, len(fields))

	colMap := map[string]string{}
	for _, c := range m.Columns {
		colMap[c.Field] = c.Name
	}

	for _, f := range fields {
		col, ok := colMap[f]
		if !ok {
			return structure{}, fmt.Errorf("field '%s' is not mapped", f)
		}

		structField, ok := ptrType.Elem().FieldByName(f)
		if !ok {
			return structure{}, fmt.Errorf("mapping fields: getting struct field %s on %T", f, m.PtrType)
		}

		colAlias := col
		if columnAliasPrefix != "" {
			colAlias = columnAliasPrefix + "_" + col
		}

		columns = append(columns, column{
			table: tbl,
			name:  col,
			alias: colAlias,
		})

		// TODO: This is inference on prefix is inaccurate, but currently correct.
		// Use nullable scan types for relation columns (when columnAliasPrefix is not empty)
		// because LEFT OUTER JOINs can return NULL for ALL columns from the joined table
		// when the foreign key is NULL. Primary table columns (empty prefix) come from
		// a direct SELECT and don't have this issue - individual nullable columns are
		// handled by pointer types.
		if columnAliasPrefix != "" {
			target = append(target, createNullableScanTarget(structField.Type))
		} else {
			target = append(target, reflect.New(structField.Type).Interface())
		}
	}

	relationList := make([]string, 0, len(relations))
	toOneRelations := map[string]joinStructure{}
	toManyRelations := map[string]joinStructure{}

	for f, rel := range relations {
		structField, ok := ptrType.Elem().FieldByName(f)
		if !ok {
			return structure{}, fmt.Errorf("mapping relations: getting struct field %s on %T", f, m.PtrType)
		}

		var subject reflect.Type
		var results map[string]joinStructure
		if structField.Type.Kind() == reflect.Ptr {
			subject = structField.Type
			results = toOneRelations
		} else if structField.Type.Kind() == reflect.Slice {
			subject = structField.Type.Elem()
			results = toManyRelations
		} else {
			return structure{}, fmt.Errorf("mapping relations: invalid struct field type %s on %T", f, m.PtrType)
		}

		otherMapping, ok := m.repo.cfg.mappings[zreflect.TypeID(subject)]
		if !ok {
			return structure{}, fmt.Errorf("no mapping available for field %s on %T", f, m.PtrType)
		}

		relMapping, ok := m.relationByField(f)
		if !ok {
			return structure{}, fmt.Errorf("unrecognized relation for field %s on %T", f, m.PtrType)
		}

		rightAlias := f
		if columnAliasPrefix != "" {
			rightAlias = columnAliasPrefix + "_" + rightAlias
		}
		rightTbl := table{
			name:  otherMapping.Table,
			alias: rightAlias,
		}

		relStructure, err := otherMapping.mapStructure(
			rightTbl,
			rightAlias,
			rel.Include.Fields,
			rel.Include.Relations,
		)
		if err != nil {
			return structure{}, fmt.Errorf("mapping structure failed for field %s on %T: %w", f, m.PtrType, err)
		}

		js := joinStructure{
			structure: relStructure,
			onPairs: zfunc.Map(zfunc.Pairs(relMapping.Columns), func(p zfunc.Pair[string, string]) [2]column {
				return [2]column{
					{
						table: tbl,
						name:  p.Key,
					},
					{
						table: rightTbl,
						name:  p.Value,
					},
				}
			}),
		}

		relationList = append(relationList, f)
		results[f] = js
	}

	return structure{
		table: tbl,

		columns: columns[pkLength:],
		fields:  fields[pkLength:],
		target:  target[pkLength:],

		primaryKey:       columns[0:pkLength],
		primaryKeyFields: fields[0:pkLength],
		primaryKeyTarget: target[0:pkLength],

		relations:       relationList,
		toOneRelations:  toOneRelations,
		toManyRelations: toManyRelations,
	}, nil
}

// lookupKey represents a key (primary or unique) that can be used for lookups.
type lookupKey struct {
	ColumnNames []string
	FieldNames  []string
	Values      []interface{}
	CanInsert   bool
	GroupKey    string // JSON-marshaled FieldNames for grouping objects by key type
	ValuesKey   string // JSON-marshaled Values for mapping results back to objects
}

func newLookupKey(columnNames, fieldNames []string, values []interface{}, canInsert bool) (lookupKey, error) {
	groupKey, err := json.Marshal(fieldNames)
	if err != nil {
		return lookupKey{}, fmt.Errorf("marshaling field names for group key: %w", err)
	}
	valuesKey, err := json.Marshal(values)
	if err != nil {
		return lookupKey{}, fmt.Errorf("marshaling values for values key: %w", err)
	}
	return lookupKey{
		ColumnNames: columnNames,
		FieldNames:  fieldNames,
		Values:      values,
		CanInsert:   canInsert,
		GroupKey:    string(groupKey),
		ValuesKey:   string(valuesKey),
	}, nil
}

// relInfo holds information about a relation for cascading Put operations.
type relInfo struct {
	fieldName      string
	relation       Relation
	parentMapping  Mapping
	relatedMapping Mapping
	fkIsLocal      bool // true if FK column(s) are on this model's table
	includeOpts    zorm.Relation
}

// copyFKValues copies FK field values between parent and related models.
func (r relInfo) copyFKValues(parentVal, relatedVal reflect.Value) error {
	for localCol, remoteCol := range r.relation.Columns {
		localFields, err := r.parentMapping.columnNamesToFields([]string{localCol})
		if err != nil {
			return fmt.Errorf("mapping local column %s: %w", localCol, err)
		}
		remoteFields, err := r.relatedMapping.columnNamesToFields([]string{remoteCol})
		if err != nil {
			return fmt.Errorf("mapping remote column %s: %w", remoteCol, err)
		}

		localField := parentVal.Elem().FieldByName(localFields[0])
		remoteField := relatedVal.Elem().FieldByName(remoteFields[0])

		if r.fkIsLocal {
			// FK is on parent: copy related's PK to parent's FK field
			// Only copy if related has a value and parent doesn't
			if !remoteField.IsZero() && localField.IsZero() {
				localField.Set(remoteField)
			}
		} else {
			// FK is on related: copy parent's PK to related's FK field
			// Only copy if parent has a value and related doesn't
			if !localField.IsZero() && remoteField.IsZero() {
				remoteField.Set(localField)
			}
		}
	}

	return nil
}
