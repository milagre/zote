package zelement

type Visitor interface {
	VisitValue(e Value) error
	VisitField(e Field) error
	VisitMethod(e Method) error
}

type Element interface {
	Accept(v Visitor) error
}

type Value struct {
	Value interface{}
}

func (e Value) Accept(v Visitor) error {
	return v.VisitValue(e)
}

type Field struct {
	Name string
}

func (e Field) Accept(v Visitor) error {
	return v.VisitField(e)
}

type Method struct {
	Name   string
	Params []interface{}
}

func (e Method) Accept(v Visitor) error {
	return v.VisitMethod(e)
}
