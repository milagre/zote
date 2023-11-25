package zapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsParam(t *testing.T) {

	for name, data := range map[string]struct {
		input        string
		expected     bool
		expectedName string
	}{
		"string": {
			input:    "users",
			expected: false,
		},
		"variable": {
			input:        "{user_id}",
			expected:     true,
			expectedName: "user_id",
		},
	} {
		t.Run(name, func(t *testing.T) {
			name, ok := isParam(data.input)
			assert.Equal(t, data.expected, ok)
			assert.Equal(t, data.expectedName, name)
		})
	}

}
