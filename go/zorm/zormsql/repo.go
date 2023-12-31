package zormsql

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zelement/zsort"
	"github.com/milagre/zote/go/zfunc"
	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zreflect"
	"github.com/milagre/zote/go/zsql"
)

// Repository
type Repository struct {
	name     string
	conn     zsql.Connection
	mappings map[string]Mapping
}

func NewRepository(name string, conn zsql.Connection) *Repository {
	return &Repository{
		name:     name,
		conn:     conn,
		mappings: map[string]Mapping{},
	}
}

func (r *Repository) AddMapping(m Mapping) {
	if r.mappings == nil {
		r.mappings = map[string]Mapping{}
	}
	r.mappings[zreflect.TypeID(reflect.TypeOf(m.Type))] = m
}

func (r *Repository) Find(ctx context.Context, ptrToListOfPtrs any, opts zorm.FindOptions) error {
	targetList, modelType, err := validatePtrToListOfPtr(ptrToListOfPtrs)
	if err != nil {
		return fmt.Errorf("invalid argument to Find: %w", err)
	}

	mapping, ok := r.mappings[zreflect.TypeID(modelType)]
	if !ok {
		return fmt.Errorf("cannot find mapping for type %v", modelType)
	}

	plan, err := buildSelectQueryPlan(modelType, r, mapping, opts.Include.Fields, opts.Include.Where, opts.Include.Sort)
	if err != nil {
		return fmt.Errorf("building query plan for list: %w", err)
	}

	limit := fmt.Sprintf(" LIMIT %d OFFSET %d", targetList.Cap(), opts.Offset)

	query := plan.query(limit)

	// fmt.Printf("Q: %s\n", query)

	rows, err := r.conn.Query(ctx, query, plan.values...)
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

func (r *Repository) Get(ctx context.Context, listOfPtrs any, opts zorm.GetOptions) error {
	targetVal, modelType, err := validateListOfPtr(listOfPtrs)
	if err != nil {
		return fmt.Errorf("invalid argument to Get: %w", err)
	}

	mapping, ok := r.mappings[zreflect.TypeID(modelType)]
	if !ok {
		return fmt.Errorf("cannot find mapping for type %v", modelType)
	}

	pk, err := mapping.primaryKeyFields()
	if err != nil {
		return fmt.Errorf("mapping primary key for in clause: %w", err)
	}

	pkValues := make([][]zelement.Element, 0, targetVal.Len())
	for i := 0; i < targetVal.Len(); i++ {
		val := targetVal.Index(i)

		values := make([]zelement.Element, 0, len(pk))
		for _, f := range pk {
			values = append(values, zelem.Value(val.Elem().FieldByName(f).Interface()))
		}

		pkValues = append(pkValues, values)
	}

	where := zclause.In{
		Left:  zfunc.Map(pk, func(f string) zelement.Element { return zelem.Field(f) }),
		Right: pkValues,
	}

	plan, err := buildSelectQueryPlan(modelType, r, mapping, opts.Include.Fields, where, nil)
	if err != nil {
		return fmt.Errorf("building query plan for list: %w", err)
	}

	query := plan.query("")

	fmt.Printf("Q: %s\n", query)

	rows, err := r.conn.Query(ctx, query, plan.values...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()

	objList := zreflect.MakeAddressableSliceOf(reflect.PointerTo(modelType), 0, targetVal.Len())

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
			obj = reflect.New(modelType).Elem()
		}

		plan.load(obj)

		if isNew {
			count++
			objList.SetLen(count)
			objList.Index(count - 1).Set(obj.Addr())
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows error: %w", err)
	}

	// TODO: ensure models are going into the right objects
	for i := 0; i < targetVal.Len(); i++ {
		targetVal.Index(i).Elem().Set(objList.Index(i).Elem())
	}

	return nil
}

func validateListOfPtr(listOfPtrs any) (reflect.Value, reflect.Type, error) {
	if listOfPtrs == nil {
		return reflect.Value{}, nil, fmt.Errorf("list of pointers required: nil provided")
	}

	targetList := reflect.ValueOf(listOfPtrs)
	if targetList.Type().Kind() != reflect.Slice {
		return reflect.Value{}, nil, fmt.Errorf("list of pointers required: non-list provided")
	}

	modelPtrType := targetList.Type().Elem()
	if modelPtrType.Kind() != reflect.Ptr {
		return reflect.Value{}, nil, fmt.Errorf("list of pointers required: list of non-pointer types provided")
	}

	modelType := targetList.Type().Elem().Elem()

	return targetList, modelType, nil
}

func validatePtrToListOfPtr(ptrToListOfPtrs any) (reflect.Value, reflect.Type, error) {
	if ptrToListOfPtrs == nil {
		return reflect.Value{}, nil, fmt.Errorf("pointer to list of pointers required: nil provided")
	}

	targetVal := reflect.ValueOf(ptrToListOfPtrs)
	if targetVal.Type().Kind() != reflect.Ptr {
		return reflect.Value{}, nil, fmt.Errorf("pointer to list of pointers required: non-pointer provided")
	}

	if targetVal.Type().Elem().Kind() != reflect.Slice {
		return reflect.Value{}, nil, fmt.Errorf("pointer to list of pointers required: non-list provided")
	}

	targetList := targetVal.Elem()

	modelPtrType := targetVal.Type().Elem().Elem()
	if modelPtrType.Kind() != reflect.Ptr {
		return reflect.Value{}, nil, fmt.Errorf("pointer to list of pointers required: list of non-pointer types provided")
	}

	modelType := targetVal.Type().Elem().Elem().Elem()

	return targetList, modelType, nil
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

func buildSelectQueryPlan(t reflect.Type, r *Repository, mapping Mapping, fields []string, clause zclause.Clause, sorts []zsort.Sort) (*selectQueryPlan, error) {
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
