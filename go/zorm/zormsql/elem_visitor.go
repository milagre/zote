package zormsql

import (
	"fmt"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zsql"
)

var _ zelement.Visitor = &elemVisitor{}

type elemVisitor struct {
	driver            zsql.Driver
	tableAlias        string
	columnAliasPrefix string
	mapping           Mapping

	// used during visits
	result string
	values []interface{}
}

func (v *elemVisitor) Visit(e zelement.Element) (string, []interface{}, error) {
	v.result = ""
	v.values = []interface{}{}

	err := e.Accept(v)
	if err != nil {
		return "", nil, fmt.Errorf("visiting element: %w", err)
	}

	return v.result, v.values, nil
}

func (v *elemVisitor) VisitValue(e zelement.Value) error {
	v.result += "?"
	v.values = append(v.values, e.Value)
	return nil
}

func (v *elemVisitor) VisitField(e zelement.Field) error {
	result, _, err := v.mapping.mapField(v.driver, v.tableAlias, v.columnAliasPrefix, e.Name)
	if err != nil {
		return fmt.Errorf("visiting field: %w", err)
	}

	v.result += result

	return nil
}

func (v *elemVisitor) VisitMethod(e zelement.Method) error {
	return fmt.Errorf("methods not supported")
}
