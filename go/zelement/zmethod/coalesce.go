package zmethod

import (
	"fmt"

	"github.com/milagre/zote/go/zelement"
)

const Coalesce Method = "coalesce"

func NewCoalesce(elems ...zelement.Element) zelement.Method {
	return zelement.Method{
		Name:   string(Coalesce),
		Params: elems,
	}
}

type coalesceValidator struct{}

func (v coalesceValidator) Validate(params []zelement.Element) error {
	if len(params) < 2 {
		return fmt.Errorf("method 'coalesce' requires at least 2 arguments")
	}
	return nil
}
