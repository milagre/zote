package zormsql

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zsort"
	"github.com/milagre/zote/go/zsql"
)

func buildSelectQueryPlan(r *Queryer, mapping Mapping, fields []string, clause zclause.Clause, sorts []zsort.Sort, limit int, offset int) (*selectQueryPlan, error) {
	if len(fields) == 0 {
		fields = mapping.allFields()
	}

	primaryTable := table{
		name:  mapping.Table,
		alias: "_1",
	}

	// primary key
	primaryKeyColumns, primaryKeyTarget, err := mapping.mappedPrimaryKeyColumns(r.conn.Driver(), primaryTable, "")
	if err != nil {
		return nil, fmt.Errorf("mapping primary key columns: %w", err)
	}

	// Columns
	columns, target, err := mapping.mapFields(r.conn.Driver(), primaryTable, "", fields)
	if err != nil {
		return nil, fmt.Errorf("mapping select columns: %w", err)
	}

	// Joins
	joins := []string{}

	// Order
	var order string
	var orders []string
	var orderValues []interface{}
	for _, s := range sorts {
		sv := sortVisitor{
			driver:  r.conn.Driver(),
			table:   primaryTable,
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
			table:   primaryTable,
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
		table: primaryTable,
		joins: joins,

		primaryKeyColumns: primaryKeyColumns,
		columns:           columns,
		fields:            fields,

		order:       order,
		orderValues: orderValues,
		where:       where,
		whereValues: whereValues,
		limit:       limit,
		offset:      offset,

		primaryKeyTarget: primaryKeyTarget,
		target:           target,
	}, nil
}

type selectQueryPlan struct {
	table             table
	joins             []string
	primaryKeyColumns []column
	columns           []column
	fields            []string
	order             string
	orderValues       []interface{}
	where             string
	whereValues       []interface{}

	limit  int
	offset int

	primaryKeyTarget []interface{}
	target           []interface{}
}

func (plan selectQueryPlan) load(v reflect.Value) {
	for i, f := range plan.fields {
		v.FieldByName(f).Set(reflect.ValueOf(plan.target[i]).Elem())
	}
}

func (plan selectQueryPlan) scannedPrimaryKey() (string, error) {
	data, err := json.Marshal(plan.primaryKeyTarget)
	return string(data), err
}

func (plan selectQueryPlan) query(driver zsql.Driver) (string, []interface{}) {
	resultColumns := make([]string, 0, len(plan.primaryKeyColumns)+len(plan.columns))

	for _, c := range append(plan.primaryKeyColumns, plan.columns...) {
		resultColumns = append(resultColumns, c.escaped(driver)+" AS "+driver.EscapeColumn(c.alias))
	}

	limitClause := fmt.Sprintf("LIMIT %d OFFSET %d", plan.limit, plan.offset)

	result := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s AS %s
		/*where: */ %s 
		/*order: */ %s
		/*limit: */ %s
	`,
		strings.Join(resultColumns, ", "),
		plan.table.escaped(driver),
		plan.table.escapedAlias(driver),
		plan.where,
		plan.order,
		limitClause,
	)

	values := append(plan.whereValues, plan.orderValues...)

	return result, values
}
