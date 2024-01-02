package zormsql

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zsort"
	"github.com/milagre/zote/go/zfunc"
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
	orders, err := zfunc.MapE(sorts, func(s zsort.Sort) (string, error) {
		dir := "ASC"
		if s.Direction == zsort.Desc {
			dir = "DESC"
		}

		col, _, err := mapping.mapField(r.conn.Driver(), firstTableAlias, "", s.Field.Name)
		if err != nil {
			return "", fmt.Errorf("mapping sort column: %w", err)
		}

		return col + " " + dir, nil
	})
	if err != nil {
		return nil, fmt.Errorf("mapping sort: %w", err)
	}
	if len(orders) > 0 {
		order = "ORDER BY " + strings.Join(orders, ", ")
	}

	// Where
	var where string
	var values []interface{}
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
		values = v
	}

	return &selectQueryPlan{
		joins: joins,

		primaryKeyColumns: primaryKeyColumns,
		columns:           columns,
		fields:            fields,

		order:  order,
		where:  where,
		values: values,

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
	where             string
	values            []interface{}

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

func (plan selectQueryPlan) query(limit string) string {
	return fmt.Sprintf(`
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
}
