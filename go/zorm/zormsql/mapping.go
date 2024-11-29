package zormsql

import (
	"fmt"
	"reflect"

	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zsql"
)

type Mapping struct {
	PtrType    interface{}
	Table      string
	PrimaryKey []string
	UniqueKeys [][]string
	Columns    []Column
	Relations  []Relation
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

func (m Mapping) hasValues(objPtr reflect.Value, fields []string) bool {
	for _, f := range fields {
		if objPtr.Elem().FieldByName(f).IsZero() {
			return false
		}
	}
	return true
}

func (m Mapping) escapedTable(driver zsql.Driver) string {
	return driver.EscapeTable(m.Table)
}

func (m Mapping) allFields() []string {
	result := make([]string, 0, len(m.Columns))
	for _, c := range m.Columns {
		result = append(result, c.Field)
	}
	return result
}

func (m Mapping) insertFields() []string {
	result := make([]string, 0, len(m.Columns))
	for _, c := range m.Columns {
		if !c.NoInsert {
			result = append(result, c.Field)
		}
	}
	return result
}

func (m Mapping) updateFields() []string {
	result := make([]string, 0, len(m.Columns))
	for _, c := range m.Columns {
		if !c.NoUpdate {
			result = append(result, c.Field)
		}
	}
	return result
}

func (m Mapping) primaryKeyFields() ([]string, error) {
	result := make([]string, 0, len(m.PrimaryKey))
	for _, pkCol := range m.PrimaryKey {
		for _, col := range m.Columns {
			if pkCol == col.Name {
				result = append(result, col.Field)
				break
			}
		}
	}

	if len(result) != len(m.PrimaryKey) {
		return nil, fmt.Errorf("primary key not fully mapped for %T", m.PtrType)
	}

	return result, nil
}

func (m Mapping) isNew(objPtr reflect.Value) {

}

func (m Mapping) mapField(driver zsql.Driver, table table, columnAliasPrefix string, field string) (column, interface{}, error) {
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

func (m Mapping) mapStructure(table table, columnAliasPrefix string, fields []string, relations zorm.Relations) (structure, error) {
	res := structure{}

	columns := make([]column, 0, len(fields))
	target := make([]interface{}, 0, len(fields))

	colMap := map[string]string{}
	for _, c := range m.Columns {
		colMap[c.Field] = c.Name
	}

	for _, f := range fields {
		col, ok := colMap[f]
		if !ok {
			return res, fmt.Errorf("field '%s' is not mapped", f)
		}
		if columnAliasPrefix != "" {
			col = columnAliasPrefix + "_" + col
		}

		structField, ok := reflect.TypeOf(m.PtrType).Elem().FieldByName(f)
		if !ok {
			return res, fmt.Errorf("mapping fields: getting struct field %s on %T", f, m.PtrType)
		}

		columns = append(columns, column{
			table: table,
			name:  col,
			alias: col,
		})
		target = append(target, reflect.New(structField.Type).Interface())
	}

	return structure{
		columns:         columns,
		target:          target,
		fields:          fields,
		relations:       []string{},
		toOneRelations:  map[string]structure{},
		toManyRelations: map[string]structure{},
	}, nil
}

func (m Mapping) mappedPrimaryKeyColumns(table table, columnAliasPrefix string) (structure, error) {
	fields, err := m.primaryKeyFields()
	if err != nil {
		return structure{}, fmt.Errorf("mapping primary key fields: %w", err)
	}
	return m.mapStructure(table, columnAliasPrefix, fields, zorm.Relations{})
}
