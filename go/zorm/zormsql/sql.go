package zormsql

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/milagre/zote/go/zelement/zsort"
	"github.com/milagre/zote/go/zfunc"
	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zreflect"
)

func find(ctx context.Context, source *Source, target interface{}, opts zorm.FindOptions) error {
	if target == nil {
		return fmt.Errorf("cannot find with a nil result target")
	}

	targetVal := reflect.ValueOf(target)
	if targetVal.Type().Kind() != reflect.Ptr {
		return fmt.Errorf("cannot find with a non-pointer result target")
	}

	if targetVal.Type().Elem().Kind() != reflect.Slice {
		return fmt.Errorf("cannot find with a non-list result target")
	}

	targetList := targetVal.Elem()

	modelPtrType := targetVal.Type().Elem().Elem()
	if modelPtrType.Kind() != reflect.Ptr {
		return fmt.Errorf("cannot find with result list of non-pointer models")
	}

	modelType := targetVal.Type().Elem().Elem().Elem()

	mapping, ok := source.mappings[zreflect.TypeID(modelType)]
	if !ok {
		return fmt.Errorf("cannot find mapping for type %v", modelType)
	}

	plan, err := buildSelectQueryPlan(modelType, source, mapping, opts.Include)
	if err != nil {
		return fmt.Errorf("building query plan for list: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s
		%s
		%s
		LIMIT %d
		OFFSET %d
	`,
		strings.Join(append(plan.primaryKeyColumns, plan.columns...), ", "),
		strings.Join(plan.joins, " LEFT JOIN "),
		plan.where,
		plan.sort,
		targetList.Cap(),
		opts.Offset,
	)

	// fmt.Printf("Q: %s\n", query)

	rows, err := source.conn.Query(ctx, query, plan.values...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()

	var obj reflect.Value
	var count int
	var currentPrimaryKey string
	scanTarget := append(plan.primaryKeyTarget, plan.target...)
	for rows.Next() {
		err := rows.Scan(scanTarget...)
		if err != nil {
			return fmt.Errorf("scanning result row: %w", err)
		}

		isNew := false
		newPrimaryKey, err := plan.scannedPrimaryKey()
		if err != nil {
			return fmt.Errorf("creating primary slug: %w", err)
		}

		if count == 0 || newPrimaryKey != currentPrimaryKey {
			isNew = true
			obj = reflect.New(targetList.Type().Elem().Elem()).Elem()
		}

		plan.load(obj)

		if isNew {
			count++
			targetList.SetLen(count)
			targetList.Index(count - 1).Set(obj.Addr())
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows error: %w", err)
	}

	return nil
}

type selectQueryPlan struct {
	joins             []string
	primaryKeyColumns []string
	columns           []string
	fields            []string
	sort              string
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

func buildSelectQueryPlan(t reflect.Type, source *Source, mapping Mapping, inc zorm.Include) (*selectQueryPlan, error) {
	fields := inc.Fields
	if len(fields) == 0 {
		fields = mapping.allFields()
	}

	firstTableAlias := mapping.Table
	firstTableAliasEscaped := source.conn.Driver().EscapeTable(firstTableAlias)

	// primary key
	primaryKeyColumns, primaryKeyTarget, err := mapping.mappedPrimaryKeyColumns(source, firstTableAlias, "")
	if err != nil {
		return nil, fmt.Errorf("mapping primary key columns: %w", err)
	}

	// Columns
	columns, target, err := mapping.mapFields(source, firstTableAlias, "", fields)
	if err != nil {
		return nil, fmt.Errorf("mapping select columns: %w", err)
	}

	// Joins
	joins := []string{
		mapping.escapedTable(source) + " AS " + firstTableAliasEscaped,
	}

	// Order
	var sort string
	sorts, err := zfunc.MapE(inc.Sort, func(s zsort.Sort) (string, error) {
		dir := "ASC"
		if s.Direction == zsort.Desc {
			dir = "DESC"
		}

		col, _, err := mapping.mapField(source, firstTableAlias, "", s.Field.Name)
		if err != nil {
			return "", fmt.Errorf("mapping sort column: %w", err)
		}

		return col + " " + dir, nil
	})
	if err != nil {
		return nil, fmt.Errorf("mapping sort: %w", err)
	}
	if len(sorts) > 0 {
		sort = "ORDER BY " + strings.Join(sorts, ", ")
	}

	// Where
	var where string
	var values []interface{}
	if inc.Where != nil {
		visitor := &whereVisitor{
			source:     source,
			tableAlias: firstTableAlias,
			mapping:    mapping,
		}
		w, v, err := visitor.Visit(inc.Where)
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

		sort:   sort,
		where:  where,
		values: values,

		primaryKeyTarget: primaryKeyTarget,
		target:           target,
	}, nil
}
