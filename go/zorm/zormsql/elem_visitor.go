package zormsql

import (
	"fmt"
	"strings"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zsql"
)

var _ zelement.Visitor = &elemVisitor{}

type elemVisitor struct {
	driver            zsql.Driver
	table             table
	columnAliasPrefix string
	mapping           Mapping
	cfg               *Config // For accessing relation mappings

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
	result, err := v.visitField(e)
	if err != nil {
		return fmt.Errorf("visiting field: %w", err)
	}

	v.result += result
	return nil
}

func (v *elemVisitor) visitField(e zelement.Field) (string, error) {
	if strings.Contains(e.Name, ".") {
		return v.visitDotDelimitedField(e.Name)
	}

	col, _, err := v.mapping.mapField(v.table, v.columnAliasPrefix, e.Name)
	if err != nil {
		return "", fmt.Errorf("visiting field: %w", err)
	}

	return col.escaped(v.driver), nil
}

func (v *elemVisitor) visitDotDelimitedField(path string) (string, error) {
	col, err := resolveDotDelimitedField(v.cfg, v.mapping, v.table, path)
	if err != nil {
		return "", err
	}
	return col.escaped(v.driver), nil
}

func (v *elemVisitor) VisitMethod(e zelement.Method) error {
	return fmt.Errorf("methods not supported")
}
