package zmethod

import (
	"fmt"

	"github.com/milagre/zote/go/zelement"
)

type methodValidator interface {
	Validate(params []zelement.Element) error
}

type Method string

var validators = map[Method]methodValidator{
	Now:           nowValidator{},
	Match:         matchValidator{},
	Contains:      containsValidator{},
	RegexpReplace: regexpReplaceValidator{},
}

func ValidateParams(m zelement.Method) error {
	validator, ok := validators[Method(m.Name)]
	if !ok {
		return fmt.Errorf("unknown method: %s", m.Name)
	}

	return validator.Validate(m.Params)
}
