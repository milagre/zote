package zormsql

import (
	"fmt"
	"reflect"

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

func (m Mapping) mapStructure(tbl table, columnAliasPrefix string, fields []string, relations zorm.Relations) (structure, error) {
	ptrType := reflect.TypeOf(m.PtrType)

	if len(fields) == 0 {
		fields = m.allFields()
	}

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
		target = append(target, reflect.New(structField.Type).Interface())
	}

	toOne := []string{}
	toMany := []string{}
	for f, _ := range relations {
		structField, ok := ptrType.Elem().FieldByName(f)
		if !ok {
			return structure{}, fmt.Errorf("mapping relations: getting struct field %s on %T", f, m.PtrType)
		}

		if structField.Type.Kind() == reflect.Slice {
			toMany = append(toMany, f)
		} else if structField.Type.Kind() == reflect.Ptr {
			toOne = append(toOne, f)
		} else {
			return structure{}, fmt.Errorf("mapping relations: invalid struct field type %s on %T", f, m.PtrType)
		}
	}

	relationList := make([]string, 0, len(relations))

	toOneRelations := map[string]joinStructure{}
	for _, f := range toOne {
		found := false
		var relMapping Relation
		for _, r := range m.Relations {
			if r.Field == f {
				relMapping = r
				found = true
				break
			}
		}
		if !found {
			return structure{}, fmt.Errorf("mapping relations: unrecognized relation for field %s on %T", f, m.PtrType)
		}

		structField, _ := ptrType.Elem().FieldByName(f)
		otherMapping, ok := m.repo.cfg.mappings[zreflect.TypeID(structField.Type)]
		if !ok {
			return structure{}, fmt.Errorf("mapping relations: no mapping available for field %s on %T", f, m.PtrType)
		}
		rel := relations[f]

		rightAlias := f
		if columnAliasPrefix != "" {
			rightAlias = columnAliasPrefix + "_" + rightAlias
		}
		rightTbl := table{
			name:  otherMapping.Table,
			alias: f,
		}

		relStructure, err := otherMapping.mapStructure(
			rightTbl,
			rightAlias,
			rel.Include.Fields,
			rel.Include.Relations,
		)
		if err != nil {
			return structure{}, fmt.Errorf("mapping relations: mapping structure failed for field %s on %T: %w", f, m.PtrType, err)
		}

		relationList = append(relationList, f)
		toOneRelations[f] = joinStructure{
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
	}

	return structure{
		table:           tbl,
		columns:         columns,
		target:          target,
		fields:          fields,
		relations:       relationList,
		toOneRelations:  toOneRelations,
		toManyRelations: map[string]joinStructure{},
	}, nil
}

func (m Mapping) mappedPrimaryKeyColumns(table table, columnAliasPrefix string) (structure, error) {
	fields, err := m.primaryKeyFields()
	if err != nil {
		return structure{}, fmt.Errorf("mapping primary key fields: %w", err)
	}
	return m.mapStructure(table, columnAliasPrefix, fields, zorm.Relations{})
}
