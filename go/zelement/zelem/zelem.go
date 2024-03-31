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

func Eq(left zelement.Element, right zelement.Element) zclause.Eq {
	return zclause.Eq{Left: left, Right: right}
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
	return zelement.Method{
		Name:   string(zmethod.Now),
		Params: []zelement.Element{},
	}
}

func MethodMatch(field string, search string) zelement.Method {
	return zelement.Method{
		Name: string(zmethod.Match),
		Params: []zelement.Element{
			Field(field),
			Value(search),
		},
	}
}
