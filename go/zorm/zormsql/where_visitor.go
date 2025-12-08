package zormsql

import (
	"fmt"
	"strings"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zsql"
)

var (
	_ zclause.Visitor  = &whereVisitor{}
	_ zelement.Visitor = &whereVisitor{}
)

type whereVisitor struct {
	driver            zsql.Driver
	table             table
	columnAliasPrefix string
	mapping           Mapping

	// used during visits
	result string
	values []interface{}
}

func (v *whereVisitor) Visit(c zclause.Clause) (string, []interface{}, error) {
	v.result = ""
	v.values = []interface{}{}

	err := c.Accept(v)
	if err != nil {
		return "", nil, fmt.Errorf("visiting clauses: %w", err)
	}

	result, values := v.result, v.values

	v.result = ""
	v.values = []interface{}{}

	return strings.TrimSpace(result), values, nil
}

func (v *whereVisitor) visitBinaryLeaf(operator string, c zclause.BinaryLeaf) error {
	err := c.Left.Accept(v)
	if err != nil {
		return fmt.Errorf("visiting binary leaf left side: %w", err)
	}

	v.result += " " + operator + " "

	err = c.Right.Accept(v)
	if err != nil {
		return fmt.Errorf("visiting binary leaf right side: %w", err)
	}

	return nil
}

func (v *whereVisitor) VisitEq(c zclause.Eq) error {
	return v.visitBinaryLeaf(v.driver.NullSafeEqualityOperator(), zclause.BinaryLeaf(c))
}

func (v *whereVisitor) VisitNeq(c zclause.Neq) error {
	return v.VisitNot(zclause.Not{
		Clause: zclause.Eq(c),
	})
}

func (v *whereVisitor) VisitGt(c zclause.Gt) error {
	return v.visitBinaryLeaf(">", zclause.BinaryLeaf(c))
}

func (v *whereVisitor) VisitGte(c zclause.Gte) error {
	return v.visitBinaryLeaf(">=", zclause.BinaryLeaf(c))
}

func (v *whereVisitor) VisitLt(c zclause.Lt) error {
	return v.visitBinaryLeaf("<", zclause.BinaryLeaf(c))
}

func (v *whereVisitor) VisitLte(c zclause.Lte) error {
	return v.visitBinaryLeaf("<=", zclause.BinaryLeaf(c))
}

func (v *whereVisitor) VisitNot(c zclause.Not) error {
	v.result += "NOT ("

	err := c.Clause.Accept(v)
	if err != nil {
		return fmt.Errorf("visiting not clause: %w", err)
	}

	v.result += ")"

	return nil
}

func (v *whereVisitor) VisitAnd(c zclause.And) error {
	return v.visitNode("AND", zclause.Node(c))
}

func (v *whereVisitor) VisitOr(c zclause.Or) error {
	return v.visitNode("OR", zclause.Node(c))
}

func (v *whereVisitor) VisitIn(c zclause.In) error {
	if len(c.Right) == 0 {
		// An empty value list cannot produce results, but isn't an invalid query
		v.result += "FALSE /* empty IN clause */"
		return nil
	}

	for _, list := range c.Right {
		if len(list) != len(c.Left) {
			return fmt.Errorf("cannot visit in clause with mismatched left and right lengths")
		}
	}

	v.result += "("

	for _, left := range c.Left {
		err := left.Accept(v)
		if err != nil {
			return fmt.Errorf("visiting left side of in clause: %w", err)
		}
	}

	v.result += ") IN ("

	for i, right := range c.Right {
		v.result += "("

		for j, elem := range right {
			err := elem.Accept(v)
			if err != nil {
				return fmt.Errorf("visiting right side of in clause: %w", err)
			}

			if j != len(right)-1 {
				v.result += ","
			}
		}

		v.result += ")"

		if i != len(c.Right)-1 {
			v.result += ","
		}
	}

	v.result += ")"

	return nil
}

func (v *whereVisitor) VisitTruthy(c zclause.Truthy) error {
	return c.Elem.Accept(v)
}

func (v *whereVisitor) VisitValue(e zelement.Value) error {
	v.result += "?"
	v.values = append(v.values, e.Value)
	return nil
}

func (v *whereVisitor) VisitField(e zelement.Field) error {
	result, err := v.visitField(e)
	if err != nil {
		return fmt.Errorf("visiting field: %w", err)
	}

	v.result += result

	return nil
}

func (v *whereVisitor) visitField(e zelement.Field) (string, error) {
	col, _, err := v.mapping.mapField(v.table, v.columnAliasPrefix, e.Name)
	result := col.escapedAlias(v.driver)
	return result, err
}

func (v *whereVisitor) VisitMethod(e zelement.Method) error {
	strp := v.driver.PrepareMethod(e.Name)

	if strp != nil {
		clause := *strp

		for i, c := range e.Params {
			subVisitor := whereVisitor{
				driver:            v.driver,
				mapping:           v.mapping,
				table:             v.table,
				columnAliasPrefix: v.columnAliasPrefix,
			}
			var err error
			if f, ok := c.(zelement.Field); ok {
				err = subVisitor.VisitField(f)
			} else if val, ok := c.(zelement.Value); ok {
				err = subVisitor.VisitValue(val)
			} else if m, ok := c.(zelement.Method); ok {
				err = subVisitor.VisitMethod(m)
			} else {
				return fmt.Errorf("visiting unrecognized element in method '%s' at param %d: %T", e.Name, i, c)
			}

			if err != nil {
				return fmt.Errorf("visiting element in method '%s' at param %d: %w", e.Name, i, err)
			}

			if strings.Contains(clause, "%s") {
				clause = strings.Replace(clause, "%s", subVisitor.result, 1)
				v.values = append(v.values, subVisitor.values...)
			} else {
				clause += subVisitor.result
				v.values = append(v.values, subVisitor.values...)
			}
		}

		v.result += clause
	} else {
		v.result += e.Name
		v.result += "("
		for i, p := range e.Params {
			err := p.Accept(v)
			if err != nil {
				return fmt.Errorf("visiting element in method '%s' at param %d: %w", e.Name, i, err)
			}
			if i < len(e.Params)-1 {
				v.result += ", "
			}
		}
		v.result += ")"
	}

	return nil
}

func (v *whereVisitor) visitNode(joiner string, c zclause.Node) error {
	if len(c.Clauses) == 0 {
		return nil
	}

	v.result += "("

	for i, child := range c.Clauses {
		err := child.Accept(v)
		if err != nil {
			return fmt.Errorf("visiting not clause: %w", err)
		}
		if i < len(c.Clauses)-1 {
			v.result += " " + joiner + " "
		}
	}

	v.result += ")"

	return nil
}
