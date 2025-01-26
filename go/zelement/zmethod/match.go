package zmethod

import (
	"fmt"

	"github.com/milagre/zote/go/zelement"
)

const Match Method = "match"

func NewMatch(fieldName string, searchValue interface{}) zelement.Method {
	return zelement.Method{
		Name: string(Match),
		Params: []zelement.Element{
			zelement.Field{Name: fieldName},
			zelement.Value{Value: searchValue},
		},
	}
}

type matchValidator struct{}

func (v matchValidator) Validate(params []zelement.Element) error {
	if len(params) != 2 {
		return fmt.Errorf("method 'match' requires two arguments")
	}

	if _, ok := params[0].(zelement.Field); !ok {
		return fmt.Errorf("method 'match' requires a field in argument 1 of 2")
	}

	if _, ok := params[1].(zelement.Value); !ok {
		return fmt.Errorf("method 'match' requires a value in argument 2 of 2")
	}

	return nil
}
