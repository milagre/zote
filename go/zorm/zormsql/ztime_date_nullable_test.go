package zormsql

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/ztime"
)

func TestConvertNullableValue_NullStringToZtimeDatePtr(t *testing.T) {
	t.Parallel()

	ns := &sql.NullString{String: "1985-08-26", Valid: true}
	got, ok := convertNullableValue(ns, reflect.TypeFor[*ztime.Date]())
	require.True(t, ok, "expected successful conversion")
	require.Equal(t, "1985-08-26", got.Interface().(*ztime.Date).String())
}

func TestCreateNullableScanTarget_ZtimeDatePtr(t *testing.T) {
	t.Parallel()

	slot := createNullableScanTarget(reflect.TypeFor[*ztime.Date]())
	_, ok := slot.(*sql.NullTime)
	require.True(t, ok, "expected NullTime scan target for *ztime.Date")
}

func TestConvertNullableValue_NullTimeToZtimeDatePtr(t *testing.T) {
	t.Parallel()

	nt := &sql.NullTime{Time: time.Date(1985, 8, 26, 0, 0, 0, 0, time.UTC), Valid: true}
	got, ok := convertNullableValue(nt, reflect.TypeFor[*ztime.Date]())
	require.True(t, ok, "expected successful conversion")
	require.Equal(t, "1985-08-26", got.Interface().(*ztime.Date).String())
}
