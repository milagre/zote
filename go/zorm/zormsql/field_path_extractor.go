package zormsql

import (
	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zsort"
)

var (
	_ zelement.Visitor = &fieldPathExtractor{}
	_ zclause.Visitor  = &fieldPathExtractor{}
)

// fieldPathExtractor extracts all field paths (including dot-delimited ones) from a clause
type fieldPathExtractor struct {
	paths []string
}

func extractFieldPaths(clause zclause.Clause) []string {
	extractor := &fieldPathExtractor{
		paths: []string{},
	}
	clause.Accept(extractor)
	return extractor.paths
}

func extractFieldPathsFromSorts(sorts []zsort.Sort) []string {
	extractor := &fieldPathExtractor{
		paths: []string{},
	}
	for _, s := range sorts {
		s.Element.Accept(extractor)
	}
	return extractor.paths
}

func (e *fieldPathExtractor) VisitEq(c zclause.Eq) error {
	c.Left.Accept(e)
	c.Right.Accept(e)
	return nil
}

func (e *fieldPathExtractor) VisitNeq(c zclause.Neq) error {
	c.Left.Accept(e)
	c.Right.Accept(e)
	return nil
}

func (e *fieldPathExtractor) VisitGt(c zclause.Gt) error {
	c.Left.Accept(e)
	c.Right.Accept(e)
	return nil
}

func (e *fieldPathExtractor) VisitGte(c zclause.Gte) error {
	c.Left.Accept(e)
	c.Right.Accept(e)
	return nil
}

func (e *fieldPathExtractor) VisitLt(c zclause.Lt) error {
	c.Left.Accept(e)
	c.Right.Accept(e)
	return nil
}

func (e *fieldPathExtractor) VisitLte(c zclause.Lte) error {
	c.Left.Accept(e)
	c.Right.Accept(e)
	return nil
}

func (e *fieldPathExtractor) VisitNot(c zclause.Not) error {
	c.Clause.Accept(e)
	return nil
}

func (e *fieldPathExtractor) VisitAnd(c zclause.And) error {
	for _, child := range c.Clauses {
		child.Accept(e)
	}
	return nil
}

func (e *fieldPathExtractor) VisitOr(c zclause.Or) error {
	for _, child := range c.Clauses {
		child.Accept(e)
	}
	return nil
}

func (e *fieldPathExtractor) VisitIn(c zclause.In) error {
	for _, left := range c.Left {
		left.Accept(e)
	}
	for _, right := range c.Right {
		for _, elem := range right {
			elem.Accept(e)
		}
	}
	return nil
}

func (e *fieldPathExtractor) VisitTruthy(c zclause.Truthy) error {
	c.Elem.Accept(e)
	return nil
}

func (e *fieldPathExtractor) VisitValue(zelement.Value) error {
	return nil
}

func (e *fieldPathExtractor) VisitField(f zelement.Field) error {
	e.paths = append(e.paths, f.Name)
	return nil
}

func (e *fieldPathExtractor) VisitMethod(m zelement.Method) error {
	for _, param := range m.Params {
		param.Accept(e)
	}
	return nil
}
