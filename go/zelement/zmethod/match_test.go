package zmethod

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/zelement"
)

func TestNewMatch(t *testing.T) {
	method := NewMatch("field1", "value1")
	assert.Equal(t, string(Match), method.Name)
	assert.Len(t, method.Params, 2)
	assert.Equal(t, zelement.Field{Name: "field1"}, method.Params[0])
	assert.Equal(t, zelement.Value{Value: "value1"}, method.Params[1])
}

func TestMatchValidation(t *testing.T) {
	t.Run("valid - field and value", func(t *testing.T) {
		method := NewMatch("field1", "value1")
		err := ValidateParams(method)
		assert.NoError(t, err)
	})

	t.Run("invalid - wrong number of params", func(t *testing.T) {
		method := zelement.Method{
			Name: string(Match),
			Params: []zelement.Element{
				zelement.Field{Name: "field1"},
			},
		}
		err := ValidateParams(method)
		assert.Error(t, err)
	})

	t.Run("invalid - first param not field", func(t *testing.T) {
		method := zelement.Method{
			Name: string(Match),
			Params: []zelement.Element{
				zelement.Value{Value: "not a field"},
				zelement.Value{Value: "value1"},
			},
		}
		err := ValidateParams(method)
		assert.Error(t, err)
	})

	t.Run("invalid - second param not value", func(t *testing.T) {
		method := zelement.Method{
			Name: string(Match),
			Params: []zelement.Element{
				zelement.Field{Name: "field1"},
				zelement.Field{Name: "not a value"},
			},
		}
		err := ValidateParams(method)
		assert.Error(t, err)
	})
}
