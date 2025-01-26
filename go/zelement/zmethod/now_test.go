package zmethod

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/zelement"
)

func TestNewNow(t *testing.T) {
	method := NewNow()
	assert.Equal(t, string(Now), method.Name)
	assert.Empty(t, method.Params)
}

func TestNowValidation(t *testing.T) {
	t.Run("valid - no params", func(t *testing.T) {
		method := NewNow()
		err := ValidateParams(method)
		assert.NoError(t, err)
	})

	t.Run("invalid - with params", func(t *testing.T) {
		method := zelement.Method{
			Name: string(Now),
			Params: []zelement.Element{
				zelement.Value{Value: "something"},
			},
		}
		err := ValidateParams(method)
		assert.Error(t, err)
	})
}
