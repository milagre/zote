package zormsql

import (
	"database/sql/driver"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stringValuer is a plain type with Value on a value receiver.
type stringValuer string

func (s stringValuer) Value() (driver.Value, error) { return string(s), nil }

// mapValuer is a map type with Value on a pointer receiver only.
type mapValuer map[string]string

func (m *mapValuer) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return "ok", nil
}

// noValuer is a plain type that doesn't implement driver.Valuer at all.
type noValuer struct{ X int }

// testStruct holds every field shape we want to exercise.
type testStruct struct {
	ValueReceiverValuer   stringValuer
	PointerReceiverValuer mapValuer
	PointerField          *mapValuer
	NonValuer             noValuer
	Int                   int
	String                string
}

func TestFieldValueForSQL(t *testing.T) {
	m := mapValuer{"a": "b"}
	obj := testStruct{
		ValueReceiverValuer:   "hello",
		PointerReceiverValuer: mapValuer{"a": "b"},
		PointerField:          &m,
		NonValuer:             noValuer{X: 42},
		Int:                   7,
		String:                "hi",
	}
	v := reflect.ValueOf(&obj).Elem()

	cases := []struct {
		name      string
		field     string
		isValuer  bool
		wantValue interface{} // if non-nil, assert the result equals this
	}{
		{"value receiver Valuer", "ValueReceiverValuer", true, nil},
		{"pointer receiver Valuer on non-pointer field", "PointerReceiverValuer", true, nil},
		{"pointer field with pointer receiver Valuer", "PointerField", true, nil},
		{"non-Valuer struct", "NonValuer", false, noValuer{X: 42}},
		{"int passes through", "Int", false, 7},
		{"string passes through", "String", false, "hi"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := fieldValueForSQL(v.FieldByName(tc.field))
			_, ok := result.(driver.Valuer)
			assert.Equal(t, tc.isValuer, ok, "driver.Valuer check")
			if tc.wantValue != nil {
				assert.Equal(t, tc.wantValue, result)
			}
		})
	}
}
