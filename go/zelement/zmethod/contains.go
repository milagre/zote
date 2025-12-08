package zmethod

import (
	"fmt"

	"github.com/milagre/zote/go/zelement"
)

const Contains Method = "contains"

func NewContains(lhs, rhs zelement.Element) zelement.Method {
	return zelement.Method{
		Name:   string(Contains),
		Params: []zelement.Element{lhs, rhs},
	}
}

type containsValidator struct{}

func (v containsValidator) Validate(params []zelement.Element) error {
	if len(params) != 1 {
		return fmt.Errorf("method 'contains' requires exactly 2 arguments")
	}
	return nil
}
