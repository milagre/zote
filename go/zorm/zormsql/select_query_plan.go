package zormsql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"4d63.com/collapsewhitespace"

	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zsort"
	"github.com/milagre/zote/go/zfunc"
	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zreflect"
	"github.com/milagre/zote/go/zsql"
)

func buildSelectQueryPlan(r *queryer, mapping Mapping, fields []string, relations zorm.Relations, clause zclause.Clause, sorts []zsort.Sort, limit int, offset int) (*selectQueryPlan, error) {
	innerPrimaryTable := table{
		name:  mapping.Table,
		alias: "target",
	}

	outerTable := table{
		alias: "inner",
	}

	outerPrimaryTable := table{
		name:  mapping.Table,
		alias: "$",
	}

	// Structure
	str, err := mapping.mapStructure(outerPrimaryTable, "", fields, relations)
	if err != nil {
		return nil, fmt.Errorf("mapping select columns: %w", err)
	}

	primaryKey := str.primaryKey

	outerPrimaryKey := zfunc.Map(primaryKey, func(c column) column {
		return column{
			table: outerTable,
			name:  c.alias,
			alias: c.alias,
		}
	})

	innerPrimaryKey := zfunc.Map(outerPrimaryKey, func(c column) column {
		return column{
			table: innerPrimaryTable,
			name:  c.alias,
			alias: c.alias,
		}
	})

	// Inner joins
	innerJoins := []join{}

	// Outer joins
	outerJoins := []join{
		{
			leftTable:  outerTable,
			rightTable: outerPrimaryTable,
			onPairs: zfunc.Map(innerPrimaryKey, func(c column) [2]column {
				return [2]column{
					{
						table: outerTable,
						name:  c.alias,
					},
					{
						table: outerPrimaryTable,
						name:  c.name,
					},
				}
			}),
		},
	}

	var visit func(tbl table, s structure) error
	visit = func(tbl table, s structure) error {
		for _, f := range s.relations {
			rel, ok := s.getRelation(f)
			if !ok {
				return fmt.Errorf("%s", f)
			}

			outerJoins = append(outerJoins, join{
				leftTable:  tbl,
				rightTable: rel.structure.table,
				onPairs:    rel.onPairs,
			})

			err := visit(rel.structure.table, rel.structure)
			if err != nil {
				return fmt.Errorf("%s/%w", f, err)
			}
		}
		return nil
	}

	err = visit(outerPrimaryTable, str)
	if err != nil {
		return nil, fmt.Errorf("mapping select relations: %w", err)
	}

	// Order
	var order string
	var orders []string
	var orderValues []interface{}
	for _, s := range sorts {
		sv := sortVisitor{
			driver:  r.conn.Driver(),
			table:   innerPrimaryTable,
			mapping: mapping,
		}

		order, vals, err := sv.Visit(s)
		if err != nil {
			return nil, fmt.Errorf("visiting sort: %w", err)
		}

		orders = append(orders, order)
		orderValues = append(orderValues, vals...)
	}
	if len(orders) > 0 {
		order = "ORDER BY " + strings.Join(orders, ", ")
	}

	// Where
	var where string
	var whereValues []interface{}
	if clause != nil {
		visitor := &whereVisitor{
			driver:  r.conn.Driver(),
			table:   innerPrimaryTable,
			mapping: mapping,
		}
		w, v, err := visitor.Visit(clause)
		if err != nil {
			return nil, fmt.Errorf("visiting select where: %w", err)
		}
		if w != "" {
			where = "WHERE " + w
			whereValues = v
		}
	}

	return &selectQueryPlan{
		innerPrimaryTable: innerPrimaryTable,
		outerPrimaryTable: outerPrimaryTable,
		outerTable:        outerTable,

		innerPrimaryKey: innerPrimaryKey,
		outerPrimaryKey: outerPrimaryKey,

		innerJoins: innerJoins,
		outerJoins: outerJoins,

		structure: str,

		order:       order,
		orderValues: orderValues,
		where:       where,
		whereValues: whereValues,
		limit:       limit,
		offset:      offset,

		target: str.fullTarget(),
	}, nil
}

type selectQueryPlan struct {
	innerPrimaryTable table
	outerPrimaryTable table
	outerTable        table

	innerPrimaryKey []column
	outerPrimaryKey []column

	innerJoins []join
	outerJoins []join

	structure structure

	order       string
	orderValues []interface{}

	where       string
	whereValues []interface{}

	limit  int
	offset int

	target []interface{}
}

func (plan selectQueryPlan) process(targetList reflect.Value, rows *sql.Rows) error {
	modelPtrType := targetList.Type().Elem()

	var obj reflect.Value
	var count int
	var currentPrimaryKey string

	for rows.Next() {
		err := rows.Scan(plan.target...)
		if err != nil {
			return fmt.Errorf("scanning find result row: %w", err)
		}

		isNew := false
		newPrimaryKey, err := json.Marshal(plan.structure.primaryKeyTarget)
		if err != nil {
			return fmt.Errorf("creating find primary slug: %w", err)
		}

		if count == 0 || string(newPrimaryKey) != currentPrimaryKey {
			isNew = true
			obj = reflect.New(modelPtrType.Elem()).Elem()
		}
		currentPrimaryKey = string(newPrimaryKey)

		plan.load(obj)

		if isNew {
			count++
			targetList.SetLen(count)
			targetList.Index(count - 1).Set(obj.Addr())
		}
	}

	return nil
}

// convertNullableValue converts a nullable scan value (sql.NullString, etc.) to its actual value
// Returns the value and whether it's valid (not NULL)
func convertNullableValue(nullableVal interface{}, targetType reflect.Type) (reflect.Value, bool) {
	val := reflect.ValueOf(nullableVal)
	if !val.IsValid() || val.IsNil() {
		return reflect.Zero(targetType), false
	}

	// Handle pointer to nullable type
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Check if it's a sql.Null* type and if it's valid
	switch v := val.Interface().(type) {
	case sql.NullString:
		if !v.Valid {
			return reflect.Zero(targetType), false
		}
		strVal := reflect.ValueOf(v.String)
		if targetType.Kind() == reflect.Ptr {
			ptrVal := reflect.New(targetType.Elem())
			ptrVal.Elem().Set(strVal)
			return ptrVal, true
		}
		return strVal, true
	case sql.NullInt64:
		if !v.Valid {
			return reflect.Zero(targetType), false
		}
		// Convert to the target type
		elemType := targetType
		isPtr := false
		if targetType.Kind() == reflect.Ptr {
			elemType = targetType.Elem()
			isPtr = true
		}
		var intVal reflect.Value
		switch elemType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal = reflect.ValueOf(v.Int64).Convert(elemType)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			intVal = reflect.ValueOf(uint64(v.Int64)).Convert(elemType)
		default:
			intVal = reflect.ValueOf(v.Int64).Convert(elemType)
		}
		if isPtr {
			ptrVal := reflect.New(elemType)
			ptrVal.Elem().Set(intVal)
			return ptrVal, true
		}
		return intVal, true
	case sql.NullFloat64:
		if !v.Valid {
			return reflect.Zero(targetType), false
		}
		elemType := targetType
		if targetType.Kind() == reflect.Ptr {
			elemType = targetType.Elem()
			floatVal := reflect.ValueOf(v.Float64).Convert(elemType)
			ptrVal := reflect.New(elemType)
			ptrVal.Elem().Set(floatVal)
			return ptrVal, true
		}
		return reflect.ValueOf(v.Float64).Convert(elemType), true
	case sql.NullBool:
		if !v.Valid {
			return reflect.Zero(targetType), false
		}
		elemType := targetType
		if targetType.Kind() == reflect.Ptr {
			elemType = targetType.Elem()
			boolVal := reflect.ValueOf(v.Bool).Convert(elemType)
			ptrVal := reflect.New(elemType)
			ptrVal.Elem().Set(boolVal)
			return ptrVal, true
		}
		return reflect.ValueOf(v.Bool).Convert(elemType), true
	case sql.NullTime:
		if !v.Valid {
			return reflect.Zero(targetType), false
		}
		elemType := targetType
		if targetType.Kind() == reflect.Ptr {
			elemType = targetType.Elem()
			timeVal := reflect.ValueOf(v.Time).Convert(elemType)
			ptrVal := reflect.New(elemType)
			ptrVal.Elem().Set(timeVal)
			return ptrVal, true
		}
		return reflect.ValueOf(v.Time).Convert(elemType), true
	default:
		if targetType.Kind() == reflect.Ptr {
			if val.Kind() == reflect.Ptr {
				return val, true
			}
			return val.Addr(), true
		}
		if val.Kind() == reflect.Ptr {
			return val.Elem(), true
		}
		return val, true
	}
}

// isNullableValueValid checks if a nullable scan value (sql.NullString, etc.) is valid (not NULL)
func isNullableValueValid(nullableVal interface{}) bool {
	val := reflect.ValueOf(nullableVal)
	if !val.IsValid() || val.IsNil() {
		return false
	}

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch v := val.Interface().(type) {
	case sql.NullString:
		return v.Valid
	case sql.NullInt64:
		return v.Valid
	case sql.NullFloat64:
		return v.Valid
	case sql.NullBool:
		return v.Valid
	case sql.NullTime:
		return v.Valid
	default:
		// Not a nullable type, assume it's valid (regular pointer from primary table)
		return true
	}
}

// isPrimaryKeyNull checks if all primary key values in the target slice are NULL
func isPrimaryKeyNull(primaryKeyTarget []interface{}) bool {
	for _, pkVal := range primaryKeyTarget {
		if isNullableValueValid(pkVal) {
			return false
		}
	}
	return true
}

func (plan selectQueryPlan) loadStructure(structure structure, offset int, v reflect.Value) (int, error) {
	var err error

	for i, f := range append(structure.primaryKeyFields, structure.fields...) {
		field := v.FieldByName(f)
		targetVal := plan.target[i+offset]
		fieldType := field.Type()

		// Try to convert from nullable type
		convertedVal, valid := convertNullableValue(targetVal, fieldType)
		if valid {
			field.Set(convertedVal)
		} else {
			// NULL value - set to zero value (nil for pointers)
			field.Set(reflect.Zero(fieldType))
		}
	}

	offset += len(structure.fields) + len(structure.primaryKeyFields)

	for _, name := range structure.relations {
		f := v.FieldByName(name)

		if rel, ok := structure.toOneRelations[name]; ok {
			relPrimaryKeyOffset := offset
			relPrimaryKeyTarget := plan.target[relPrimaryKeyOffset : relPrimaryKeyOffset+len(rel.structure.primaryKeyTarget)]

			if isPrimaryKeyNull(relPrimaryKeyTarget) {
				// Skip loading null relation and advance offset past all relation columns (including nested relations)
				offset += len(rel.structure.fullTarget())
				continue
			}

			if f.IsNil() {
				t, _ := v.Type().FieldByName(name)
				empty := reflect.New(t.Type.Elem())
				f.Set(empty)
			}

			offset, err = plan.loadStructure(rel.structure, offset, f.Elem())
			if err != nil {
				return 0, fmt.Errorf("loading relation %s data: %w", f, err)
			}
		} else {
			rel, ok = structure.toManyRelations[name]
			if !ok {
				panic("invalid relation in select query plan structure")
			}

			if f.IsNil() {
				t, _ := v.Type().FieldByName(name)
				f.Set(zreflect.MakeAddressableSliceOf(t.Type.Elem(), 0, 1))
			}

			var elem reflect.Value

			newPrimaryKeyBytes, err := json.Marshal(rel.structure.primaryKeyTarget)
			if err != nil {
				return 0, fmt.Errorf("creating relation primary slug: %w", err)
			}
			newPrimaryKey := string(newPrimaryKeyBytes)
			currentPrimaryKey := string(rel.structure.prevPrimaryKey)
			if f.Len() == 0 || newPrimaryKey != currentPrimaryKey {
				elem = reflect.New(f.Type().Elem().Elem())
				f.Set(reflect.Append(f, elem))
			} else {
				elem = f.Index(f.Len() - 1)
			}

			offset, err = plan.loadStructure(rel.structure, offset, elem.Elem())
			if err != nil {
				return 0, fmt.Errorf("loading relation %s data: %w", f, err)
			}

			rel.structure.prevPrimaryKey = newPrimaryKey
			structure.toManyRelations[name] = rel
		}

	}

	return offset, nil
}

func (plan selectQueryPlan) load(v reflect.Value) {
	plan.loadStructure(plan.structure, 0, v)
}

func (plan selectQueryPlan) query(driver zsql.Driver) (string, []interface{}) {
	outerColumns := strings.Join(
		zfunc.Map(
			plan.structure.fullColumns(),
			func(c column) string {
				return driver.EscapeTableColumn(c.table.alias, c.name)
			},
		),
		", ",
	)

	innerColumns := strings.Join(
		zfunc.Map(
			plan.innerPrimaryKey,
			func(c column) string {
				return fmt.Sprintf(
					"%s AS %s",
					driver.EscapeTableColumn(c.table.alias, c.name),
					driver.EscapeColumn(c.alias),
				)
			},
		),
		", ",
	)

	outerJoins := strings.Join(zfunc.Map(
		plan.outerJoins,
		func(j join) string {
			return fmt.Sprintf(
				`LEFT OUTER JOIN %s AS %s ON (%s)`,
				driver.EscapeTable(j.rightTable.name),
				driver.EscapeTable(j.rightTable.alias),
				strings.Join(
					zfunc.Map(j.onPairs, func(cols [2]column) string {
						return fmt.Sprintf(
							"%s=%s",
							driver.EscapeTableColumn(
								cols[0].table.alias,
								cols[0].name,
							),
							driver.EscapeTableColumn(
								cols[1].table.alias,
								cols[1].name,
							),
						)
					}),
					" AND ",
				),
			)
		},
	), " ")

	where := plan.where
	order := plan.order

	limit := fmt.Sprintf("LIMIT %d OFFSET %d", plan.limit, plan.offset)

	target := driver.EscapeTable(plan.innerPrimaryTable.name)
	targetAlias := driver.EscapeTable(plan.innerPrimaryTable.alias)
	outerAlias := driver.EscapeTable(plan.outerTable.alias)

	result := collapsewhitespace.String(fmt.Sprintf(`
		SELECT
			%s
		FROM (
			SELECT 
				%s
			FROM
				%s AS %s
			/*where*/ %s 
			/*order*/ %s
			/*limit*/ %s
		) AS %s
		%s
	`,
		outerColumns,
		innerColumns,
		target,
		targetAlias,
		where,
		order,
		limit,
		outerAlias,
		outerJoins,
	))

	values := append(plan.whereValues, plan.orderValues...)

	return result, values
}
