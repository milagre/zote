package zelem

import (
	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zsort"
)

func Field(path string) zelement.Field {
	return zelement.Field{
		Name: path,
	}
}

func Value(v interface{}) zelement.Value {
	return zelement.Value{
		Value: v,
	}
}

func And(clauses ...zclause.Clause) zclause.And {
	return zclause.And{
		Clauses: clauses,
	}
}

func Eq(left zelement.Element, right zelement.Element) zclause.Eq {
	return zclause.Eq{
		Left:  left,
		Right: right,
	}
}

func In(left []zelement.Element, right [][]zelement.Element) zclause.In {
	return zclause.In{
		Left:  left,
		Right: right,
	}
}

func Sorts(sorts ...zsort.Sort) []zsort.Sort {
	return sorts
}

func Asc(elem zelement.Element) zsort.Sort {
	return zsort.Sort{
		Direction: zsort.Asc,
		Element:   elem,
	}
}

func Desc(elem zelement.Element) zsort.Sort {
	return zsort.Sort{
		Direction: zsort.Desc,
		Element:   elem,
	}
}
