package zormsql

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// ptrScanReceiver has Scan only on the pointer receiver (like models that embed JSON).
type ptrScanReceiver struct {
	Payload string
}

func (p *ptrScanReceiver) Scan(src interface{}) error {
	switch x := src.(type) {
	case []byte:
		p.Payload = string(x)
	case string:
		p.Payload = x
	default:
		p.Payload = ""
	}
	return nil
}

func TestConvertNullableValueNullStringPointerTargetUsesScanner(t *testing.T) {
	ns := &sql.NullString{String: "hello-json", Valid: true}
	v, ok := convertNullableValue(ns, reflect.TypeFor[*ptrScanReceiver]())
	require.True(t, ok)
	require.Equal(t, reflect.Ptr, v.Kind())
	got := v.Interface().(*ptrScanReceiver)
	require.Equal(t, "hello-json", got.Payload)
}

func TestConvertNullableValueNullStringValueTargetUsesScanner(t *testing.T) {
	ns := &sql.NullString{String: "value-type", Valid: true}
	v, ok := convertNullableValue(ns, reflect.TypeFor[ptrScanReceiver]())
	require.True(t, ok)
	require.Equal(t, reflect.Struct, v.Kind())
	require.Equal(t, "value-type", v.FieldByName("Payload").String())
}
