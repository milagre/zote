package zmethod

import (
	"fmt"

	"github.com/milagre/zote/go/zelement"
)

type Method string

const (
	Now   Method = "now"
	Match Method = "match"
)

func ValidateParams(m zelement.Method) error {
	switch Method(m.Name) {
	case Now:
		if len(m.Params) != 0 {
			return fmt.Errorf("zmethod 'now' does not support arguments")
		}

	case Match:
		if len(m.Params) != 2 {
			return fmt.Errorf("zmethod 'match' requires two arguments")
		}
		if _, ok := m.Params[0].(zelement.Field); !ok {
			return fmt.Errorf("zmethod 'match' requires a field in argument 1 of 2")
		}
		if _, ok := m.Params[0].(zelement.Value); !ok {
			return fmt.Errorf("zmethod 'match' requires a value in argument 2 of 2")
		}
	}

	return nil
}
