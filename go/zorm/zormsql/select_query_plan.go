package zormsql

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zsort"
)

func buildSelectQueryPlan(r *Repository, mapping Mapping, fields []string, clause zclause.Clause, sorts []zsort.Sort) (*selectQueryPlan, error) {
	if len(fields) == 0 {
		fields = mapping.allFields()
	}

	firstTableAlias := mapping.Table
	firstTableAliasEscaped := r.conn.Driver().EscapeTable(firstTableAlias)

	// primary key
	primaryKeyColumns, primaryKeyTarget, err := mapping.mappedPrimaryKeyColumns(r.conn.Driver(), firstTableAlias, "")
	if err != nil {
		return nil, fmt.Errorf("mapping primary key columns: %w", err)
	}

	// Columns
	columns, target, err := mapping.mapFields(r.conn.Driver(), firstTableAlias, "", fields)
	if err != nil {
		return nil, fmt.Errorf("mapping select columns: %w", err)
	}

	// Joins
	joins := []string{
		mapping.escapedTable(r.conn.Driver()) + " AS " + firstTableAliasEscaped,
	}

	// Order
	var order string
	var orders []string
	var orderValues []interface{}
	for _, s := range sorts {
		sv := sortVisitor{
			driver:     r.conn.Driver(),
			tableAlias: firstTableAlias,
			mapping:    mapping,
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
			driver:     r.conn.Driver(),
			tableAlias: firstTableAlias,
			mapping:    mapping,
		}
		w, v, err := visitor.Visit(clause)
		if err != nil {
			return nil, fmt.Errorf("visiting select where: %w", err)
		}
		where = "WHERE " + w
		whereValues = v
	}

	return &selectQueryPlan{
		joins: joins,

		primaryKeyColumns: primaryKeyColumns,
		columns:           columns,
		fields:            fields,

		order:       order,
		orderValues: orderValues,
		where:       where,
		whereValues: whereValues,

		primaryKeyTarget: primaryKeyTarget,
		target:           target,
	}, nil
}

type selectQueryPlan struct {
	joins             []string
	primaryKeyColumns []string
	columns           []string
	fields            []string
	order             string
	orderValues       []interface{}
	where             string
	whereValues       []interface{}

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

func (plan selectQueryPlan) query(limit string) (string, []interface{}) {
	result := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s
		/*where: */ %s 
		/*order: */ %s
		/*limit: */ %s
	`,
		strings.Join(append(plan.primaryKeyColumns, plan.columns...), ", "),
		strings.Join(plan.joins, " LEFT JOIN "),
		plan.where,
		plan.order,
		limit,
	)

	values := append(plan.whereValues, plan.orderValues...)

	return result, values
}
