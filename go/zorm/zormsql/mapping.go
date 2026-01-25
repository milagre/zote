package zormsql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/milagre/zote/go/zfunc"
	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zreflect"
)

type Mapping struct {
	PtrType    interface{}
	Table      string
	PrimaryKey []string
	UniqueKeys [][]string
	Columns    []Column
	Relations  []Relation

	repo *Repository
}

type Column struct {
	Name     string
	Field    string
	NoInsert bool
	NoUpdate bool
}

type Relation struct {
	Table   string
	Columns map[string]string
	Field   string
}

func (m Mapping) relationByField(f string) (Relation, bool) {
	for _, r := range m.Relations {
		if r.Field == f {
			return r, true
		}
	}
	return Relation{}, false
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

// createNullableScanTarget creates a nullable scan target (sql.NullString, sql.NullInt64, etc.)
// for a given field type. This is used for relation columns that might be NULL.
func createNullableScanTarget(fieldType reflect.Type) interface{} {
	// Handle pointer types - get the underlying type
	elemType := fieldType
	if fieldType.Kind() == reflect.Ptr {
		elemType = fieldType.Elem()
	}

	switch elemType.Kind() {
	case reflect.String:
		return &sql.NullString{}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &sql.NullInt64{}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Use NullInt64 for unsigned integers (may lose precision for very large uint64)
		return &sql.NullInt64{}
	case reflect.Float32, reflect.Float64:
		return &sql.NullFloat64{}
	case reflect.Bool:
		return &sql.NullBool{}
	default:
		// For time.Time and other types, try to use the pointer type
		if elemType == reflect.TypeOf(time.Time{}) {
			return &sql.NullTime{}
		}
		// For unknown types, fall back to pointer (may fail on NULL, but that's the current behavior)
		return reflect.New(fieldType).Interface()
	}
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
