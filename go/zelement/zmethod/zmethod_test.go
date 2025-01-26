package zmethod

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/zelement"
)

func TestValidateParams(t *testing.T) {
	t.Run("invalid - unknown method", func(t *testing.T) {
		method := zelement.Method{
			Name: "unknown",
		}
		err := ValidateParams(method)
		assert.Error(t, err)
	})
}
