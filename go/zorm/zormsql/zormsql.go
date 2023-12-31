package zormsql

import (
	"fmt"
	"reflect"

	"github.com/milagre/zote/go/zsql"
)

var ErrNotFound = fmt.Errorf("not found")

// Mapping
type Mapping struct {
	Type       interface{}
	Table      string
	PrimaryKey []string
	UniqueKeys [][]string
	Columns    []Column
	Relations  []Relation
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
		return nil, fmt.Errorf("primary key not fully mapped for %T", m.Type)
	}

	return result, nil
}

func (m Mapping) mapField(driver zsql.Driver, tableAlias string, columnAliasPrefix string, field string) (string, interface{}, error) {
	for _, c := range m.Columns {
		if field == c.Field {
			col := c.Name
			if columnAliasPrefix != "" {
				col = columnAliasPrefix + "_" + col
			}

			structField, ok := reflect.TypeOf(m.Type).FieldByName(field)
			if !ok {
				return "", nil, fmt.Errorf("getting struct field %s on %T", field, m.Type)
			}

			return driver.EscapeTableColumn(tableAlias, col), reflect.New(structField.Type).Interface(), nil
		}
	}

	return "", nil, fmt.Errorf("field %s is not mapped", field)
}

func (m Mapping) mapFields(driver zsql.Driver, tableAlias string, columnAliasPrefix string, fields []string) ([]string, []interface{}, error) {
	columns := make([]string, 0, len(fields))
	target := make([]interface{}, 0, len(fields))

	colMap := map[string]string{}
	for _, c := range m.Columns {
		colMap[c.Field] = c.Name
	}

	for _, f := range fields {
		col, ok := colMap[f]
		if !ok {
			return nil, nil, fmt.Errorf("field %s is not mapped", f)
		}
		if columnAliasPrefix != "" {
			col = columnAliasPrefix + "_" + col
		}

		structField, ok := reflect.TypeOf(m.Type).FieldByName(f)
		if !ok {
			return nil, nil, fmt.Errorf("getting struct field %s on %T", f, m.Type)
		}

		columns = append(columns, driver.EscapeTableColumn(tableAlias, col))
		target = append(target, reflect.New(structField.Type).Interface())
	}

	return columns, target, nil
}

func (m Mapping) mappedPrimaryKeyColumns(driver zsql.Driver, tableAlias string, columnAliasPrefix string) ([]string, []interface{}, error) {
	result := make([]string, 0, len(m.PrimaryKey))
	target := make([]interface{}, 0, len(m.PrimaryKey))

	colMap := map[string]string{}
	for _, c := range m.Columns {
		colMap[c.Name] = c.Field
	}

	for i, col := range m.PrimaryKey {
		f, ok := colMap[col]
		if !ok {
			return nil, nil, fmt.Errorf("primary key column %s is not mapped", col)
		}
		if columnAliasPrefix != "" {
			col = columnAliasPrefix + "_" + col
		}

		structField, ok := reflect.TypeOf(m.Type).FieldByName(f)
		if !ok {
			return nil, nil, fmt.Errorf("getting struct field %s on %T", f, m.Type)
		}

		result = append(result, driver.EscapeTableColumn(tableAlias, col)+" AS "+fmt.Sprintf("_%d", i))
		target = append(target, reflect.New(structField.Type).Interface())
	}

	return result, target, nil
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
