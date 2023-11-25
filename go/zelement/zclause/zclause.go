package zclause

import (
	"github.com/milagre/zote/go/zelement/zelem"
)

type Clause interface {
	Accept(v Visitor) error
}

type Binary struct {
	Left  zelem.Element
	Right zelem.Element
}

type Unary struct {
	Elem zelem.Element
}

type Nary struct {
	Elem []zelem.Element
}

type Eq Binary

func (c Eq) Accept(v Visitor) error {
	return v.VisitEq(c)
}

type Neq Binary

func (c Neq) Accept(v Visitor) error {
	return v.VisitNeq(c)
}

type Gt Binary

func (c Gt) Accept(v Visitor) error {
	return v.VisitGt(c)
}

type Gte Binary

func (c Gte) Accept(v Visitor) error {
	return v.VisitGte(c)
}

type Lt Binary

func (c Lt) Accept(v Visitor) error {
	return v.VisitLt(c)
}

type Lte Binary

func (c Lte) Accept(v Visitor) error {
	return v.VisitLte(c)
}

type Not Unary

func (c Not) Accept(v Visitor) error {
	return v.VisitNot(c)
}

type And Nary

func (c And) Accept(v Visitor) error {
	return v.VisitAnd(c)
}

type Or Nary

func (c Or) Accept(v Visitor) error {
	return v.VisitOr(c)
}

type Visitor interface {
	VisitEq(eq Eq) error
	VisitNeq(neq Neq) error

	VisitGt(gt Gt) error
	VisitGte(gte Gte) error

	VisitLt(lt Lt) error
	VisitLte(lte Lte) error

	VisitNot(not Not) error

	VisitAnd(and And) error
	VisitOr(or Or) error
}
