package zclause

import (
	"github.com/milagre/zote/go/zelement"
)

type Clause interface {
	Accept(v Visitor) error
}

type BinaryLeaf struct {
	Left  zelement.Element
	Right zelement.Element
}

type UnaryNode struct {
	Clause Clause
}

type Node struct {
	Clauses []Clause
}

type Eq BinaryLeaf

func (c Eq) Accept(v Visitor) error {
	return v.VisitEq(c)
}

type Neq BinaryLeaf

func (c Neq) Accept(v Visitor) error {
	return v.VisitNeq(c)
}

type Gt BinaryLeaf

func (c Gt) Accept(v Visitor) error {
	return v.VisitGt(c)
}

type Gte BinaryLeaf

func (c Gte) Accept(v Visitor) error {
	return v.VisitGte(c)
}

type Lt BinaryLeaf

func (c Lt) Accept(v Visitor) error {
	return v.VisitLt(c)
}

type Lte BinaryLeaf

func (c Lte) Accept(v Visitor) error {
	return v.VisitLte(c)
}

type Not UnaryNode

func (c Not) Accept(v Visitor) error {
	return v.VisitNot(c)
}

type And Node

func (c And) Accept(v Visitor) error {
	return v.VisitAnd(c)
}

type Or Node

func (c Or) Accept(v Visitor) error {
	return v.VisitOr(c)
}

type In struct {
	Left  []zelement.Element
	Right [][]zelement.Element
}

func (c In) Accept(v Visitor) error {
	return v.VisitIn(c)
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

	VisitIn(in In) error
}
