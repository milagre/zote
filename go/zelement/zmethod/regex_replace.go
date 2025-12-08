package zmethod

import (
	"fmt"

	"github.com/milagre/zote/go/zelement"
)

const RegexpReplace Method = "regexp_replace"

func NewRegexpReplace(subject, pattern, replacement zelement.Element) zelement.Method {
	return zelement.Method{
		Name: string(RegexpReplace),
		Params: []zelement.Element{
			subject,
			pattern,
			replacement,
		},
	}
}

type regexpReplaceValidator struct{}

func (v regexpReplaceValidator) Validate(params []zelement.Element) error {
	if len(params) != 3 {
		return fmt.Errorf("method 'regexp_replace' requires exactly 3 arguments")
	}

	return nil
}
