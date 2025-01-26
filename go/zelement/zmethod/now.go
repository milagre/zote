package zmethod

import (
	"fmt"

	"github.com/milagre/zote/go/zelement"
)

const Now Method = "now"

func NewNow() zelement.Method {
	return zelement.Method{
		Name:   string(Now),
		Params: []zelement.Element{},
	}
}

type nowValidator struct{}

func (v nowValidator) Validate(params []zelement.Element) error {
	if len(params) != 0 {
		return fmt.Errorf("method 'now' does not support arguments")
	}
	return nil
}
