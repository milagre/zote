package zormsql

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zsort"
	"github.com/milagre/zote/go/zfunc"
	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zsql"
)

func buildSelectQueryPlan(r *Queryer, mapping Mapping, fields []string, relations zorm.Relations, clause zclause.Clause, sorts []zsort.Sort, limit int, offset int) (*selectQueryPlan, error) {
	if len(fields) == 0 {
		fields = mapping.allFields()
	}

	innerPrimaryTable := table{
		name:  mapping.Table,
		alias: "target",
	}

	outerTable := table{
		alias: "inner",
	}

	outerPrimaryTable := table{
		name:  mapping.Table,
		alias: "outer",
	}

	// inner primary key
	innerPrimaryKeyStructure, err := mapping.mappedPrimaryKeyColumns(innerPrimaryTable, "")
	if err != nil {
		return nil, fmt.Errorf("mapping primary key columns: %w", err)
	}

	// outer primary key
	outerPrimaryKey := zfunc.Map(innerPrimaryKeyStructure.columns, func(c column) column {
		return column{
			table: outerTable,
			name:  c.alias,
			alias: c.alias,
		}
	})

	// Structure
	structure, err := mapping.mapStructure(outerPrimaryTable, "", fields, relations)
	if err != nil {
		return nil, fmt.Errorf("mapping select columns: %w", err)
	}

	// Inner joins
	innerJoins := []join{}

	// Outer joins
	outerJoins := []join{
		{
			leftTable:  outerTable,
			rightTable: outerPrimaryTable,
			onPairs: zfunc.Map(innerPrimaryKeyStructure.columns, func(c column) [2]column {
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

		innerPrimaryKey: innerPrimaryKeyStructure.columns,
		outerPrimaryKey: outerPrimaryKey,

		innerJoins: innerJoins,
		outerJoins: outerJoins,

		structure: structure,

		order:       order,
		orderValues: orderValues,
		where:       where,
		whereValues: whereValues,
		limit:       limit,
		offset:      offset,

		primaryKeyTarget: innerPrimaryKeyStructure.target,
		target:           structure.target,
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

	primaryKeyTarget []interface{}
	target           []interface{}
}

func (plan selectQueryPlan) loadStructure(structure structure, offset int, v reflect.Value) int {
	for i, f := range structure.fields {
		v.FieldByName(f).Set(reflect.ValueOf(plan.target[i+offset]).Elem())
	}

	offset += len(structure.fields)

	for _, name := range structure.relations {
		f := v.FieldByName(name)

		if rel, ok := structure.toOneRelations[name]; ok {
			if f.IsNil() {
				t, _ := v.Type().FieldByName(name)
				empty := reflect.New(t.Type.Elem())
				f.Set(empty.Addr())
			}

			offset = plan.loadStructure(rel, offset, f)
		} else {
			rel, ok = structure.toManyRelations[name]
			if !ok {
				panic("invalid relation in select query plan structure")
			}

			if f.IsNil() {
				// TODO: this is a list
				t, _ := v.Type().FieldByName(name)
				empty := reflect.New(t.Type.Elem())
				f.Set(empty.Addr())
			}

			offset = plan.loadStructure(rel, offset, f)
		}
	}

	return offset
}

func (plan selectQueryPlan) load(v reflect.Value) {
	plan.loadStructure(plan.structure, 0, v)
}

func (plan selectQueryPlan) scannedPrimaryKey() (string, error) {
	data, err := json.Marshal(plan.primaryKeyTarget)
	return string(data), err
}

func (plan selectQueryPlan) query(driver zsql.Driver) (string, []interface{}) {
	outerColumns := strings.Join(
		zfunc.Map(
			append(plan.outerPrimaryKey, plan.structure.columns...),
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

	outerJoins := strings.Join(
		zfunc.Map(
			plan.outerJoins,
			func(j join) string {
				return fmt.Sprintf(
					`%s AS %s ON (%s)`,
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
		),
		" LEFT OUTER JOIN ",
	)

	where := plan.where
	order := plan.order

	limit := fmt.Sprintf("LIMIT %d OFFSET %d", plan.limit, plan.offset)

	target := driver.EscapeTable(plan.innerPrimaryTable.name)
	targetAlias := driver.EscapeTable(plan.innerPrimaryTable.alias)
	outerAlias := driver.EscapeTable(plan.outerTable.alias)

	result := fmt.Sprintf(`
		SELECT
			/*outerColumns*/ %s
		FROM (
			SELECT 
				/*innerColumns*/ %s
			FROM
				/*target*/ %s
				AS
				/*targetAlias*/ %s
			/*where*/ %s 
			/*order*/ %s
			/*limit*/ %s
		) AS /*outerAlias*/ %s
		LEFT OUTER JOIN
		/*outerJoins*/ %s
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
	)

	values := append(plan.whereValues, plan.orderValues...)

	return result, values
}
