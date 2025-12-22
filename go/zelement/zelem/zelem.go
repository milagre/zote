package zelem

import (
	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zmethod"
	"github.com/milagre/zote/go/zelement/zsort"
)

func Field(path string) zelement.Field {
	return zelement.Field{Name: path}
}

func Value(v interface{}) zelement.Value {
	return zelement.Value{Value: v}
}

func And(clauses ...zclause.Clause) zclause.And {
	return zclause.And{Clauses: clauses}
}

func Or(clauses ...zclause.Clause) zclause.Or {
	return zclause.Or{Clauses: clauses}
}

func Not(clause zclause.Clause) zclause.Not {
	return zclause.Not{Clause: clause}
}

func Truthy(e zelement.Element) zclause.Truthy {
	return zclause.Truthy{Elem: e}
}

func False(e zelement.Element) zclause.Not {
	return zclause.Not{Clause: zclause.Truthy{Elem: e}}
}

func Eq(left zelement.Element, right zelement.Element) zclause.Eq {
	return zclause.Eq{Left: left, Right: right}
}

func Neq(left zelement.Element, right zelement.Element) zclause.Neq {
	return zclause.Neq{Left: left, Right: right}
}

func In(left []zelement.Element, right [][]zelement.Element) zclause.In {
	return zclause.In{Left: left, Right: right}
}

func Lte(left zelement.Element, right zelement.Element) zclause.Lte {
	return zclause.Lte{Left: left, Right: right}
}

func Lt(left zelement.Element, right zelement.Element) zclause.Lt {
	return zclause.Lt{Left: left, Right: right}
}

func Gte(left zelement.Element, right zelement.Element) zclause.Gte {
	return zclause.Gte{Left: left, Right: right}
}

func Gt(left zelement.Element, right zelement.Element) zclause.Gt {
	return zclause.Gt{Left: left, Right: right}
}

func Sorts(sorts ...zsort.Sort) []zsort.Sort { return sorts }

func Asc(elem zelement.Element) zsort.Sort {
	return zsort.Sort{Direction: zsort.Asc, Element: elem}
}

func Desc(elem zelement.Element) zsort.Sort {
	return zsort.Sort{Direction: zsort.Desc, Element: elem}
}

func Empty() zclause.Clause {
	return nil
}

func MethodNow() zelement.Method {
	return zmethod.NewNow()
}

func MethodMatch(field string, search string) zelement.Method {
	return zmethod.NewMatch(field, search)
}
