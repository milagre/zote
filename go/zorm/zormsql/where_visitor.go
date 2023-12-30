package zormsql

import (
	"fmt"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zclause"
)

var _ zclause.Visitor = &whereVisitor{}
var _ zelement.Visitor = &whereVisitor{}

type whereVisitor struct {
	source            *Source
	tableAlias        string
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

	return result, values, nil
}

func (v *whereVisitor) visitBinaryLeaf(operator string, c zclause.BinaryLeaf) error {
	err := c.Left.Accept(v)
	if err != nil {
		return fmt.Errorf("visiting binary leaf left side: %w", err)
	}

	v.result += " " + operator + " "

	c.Right.Accept(v)
	if err != nil {
		return fmt.Errorf("visiting binary leaf right side: %w", err)
	}

	v.result += " "

	return nil
}

func (v *whereVisitor) VisitEq(c zclause.Eq) error {
	return v.visitBinaryLeaf(v.source.conn.Driver().NullSafeEqualityOperator(), zclause.BinaryLeaf(c))
}

func (v *whereVisitor) VisitNeq(c zclause.Neq) error {
	return v.VisitNot(zclause.Not{
		Clause: zclause.Eq{
			Left:  c.Left,
			Right: c.Right,
		},
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

	v.result += ") "

	return nil
}

func (v *whereVisitor) VisitAnd(c zclause.And) error {
	return v.visitNode("AND", zclause.Node(c))
}

func (v *whereVisitor) VisitOr(c zclause.Or) error {
	return v.visitNode("OR", zclause.Node(c))
}

func (v *whereVisitor) VisitValue(e zelement.Value) error {
	v.result += "?"
	v.values = append(v.values, e.Value)
	return nil
}

func (v *whereVisitor) VisitField(e zelement.Field) error {
	result, _, err := v.mapping.mapField(v.source, v.tableAlias, v.columnAliasPrefix, e.Name)
	if err != nil {
		return fmt.Errorf("visiting field: %w", err)
	}

	result += result

	return nil
}

func (v *whereVisitor) VisitMethod(e zelement.Method) error {
	return fmt.Errorf("methods not supported")
}

func (v *whereVisitor) visitNode(joiner string, c zclause.Node) error {
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

	v.result += ") "

	return nil
}
