package zormsql

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zutil"
)

type stringAlias string

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

func TestConvertNullableValueTable(t *testing.T) {
	t.Parallel()

	fixTime := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)

	cases := []struct {
		name       string
		nullable   interface{}
		targetType reflect.Type
		wantOK     bool
		check      func(t *testing.T, v reflect.Value)
	}{
		// --- nil / invalid nullable ---
		{
			name:       "nil interface",
			nullable:   nil,
			targetType: reflect.TypeFor[string](),
			wantOK:     false,
		},
		{
			name:       "nil NullString pointer",
			nullable:   (*sql.NullString)(nil),
			targetType: reflect.TypeFor[string](),
			wantOK:     false,
		},
		{
			name:       "NullString invalid",
			nullable:   &sql.NullString{Valid: false},
			targetType: reflect.TypeFor[string](),
			wantOK:     false,
		},
		{
			name:       "NullInt64 invalid",
			nullable:   &sql.NullInt64{Valid: false},
			targetType: reflect.TypeFor[int64](),
			wantOK:     false,
		},
		{
			name:       "NullFloat64 invalid",
			nullable:   &sql.NullFloat64{Valid: false},
			targetType: reflect.TypeFor[float64](),
			wantOK:     false,
		},
		{
			name:       "NullBool invalid",
			nullable:   &sql.NullBool{Valid: false},
			targetType: reflect.TypeFor[bool](),
			wantOK:     false,
		},
		{
			name:       "NullTime invalid",
			nullable:   &sql.NullTime{Valid: false},
			targetType: reflect.TypeFor[time.Time](),
			wantOK:     false,
		},

		// --- NullString ---
		{
			name:       "NullString to string",
			nullable:   &sql.NullString{String: "plain", Valid: true},
			targetType: reflect.TypeFor[string](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, "plain", v.String())
			},
		},
		{
			name:       "NullString to named string type (ConvertibleTo)",
			nullable:   &sql.NullString{String: "alias", Valid: true},
			targetType: reflect.TypeFor[stringAlias](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, stringAlias("alias"), v.Interface().(stringAlias))
			},
		},
		{
			name:       "NullString to *string",
			nullable:   &sql.NullString{String: "heap", Valid: true},
			targetType: reflect.TypeFor[*string](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, reflect.Ptr, v.Kind())
				require.Equal(t, "heap", v.Elem().String())
			},
		},
		{
			name:       "NullString to *ptrScanReceiver via Scanner",
			nullable:   &sql.NullString{String: "hello-json", Valid: true},
			targetType: reflect.TypeFor[*ptrScanReceiver](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				got := v.Interface().(*ptrScanReceiver)
				require.Equal(t, "hello-json", got.Payload)
			},
		},
		{
			name:       "NullString to ptrScanReceiver value via Scanner",
			nullable:   &sql.NullString{String: "value-type", Valid: true},
			targetType: reflect.TypeFor[ptrScanReceiver](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, reflect.Struct, v.Kind())
				require.Equal(t, "value-type", v.FieldByName("Payload").String())
			},
		},

		// --- NullInt64 ---
		{
			name:       "NullInt64 to int64",
			nullable:   &sql.NullInt64{Int64: 42, Valid: true},
			targetType: reflect.TypeFor[int64](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, int64(42), v.Int())
			},
		},
		{
			name:       "NullInt64 to *int64",
			nullable:   &sql.NullInt64{Int64: -7, Valid: true},
			targetType: reflect.TypeFor[*int64](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, int64(-7), v.Elem().Int())
			},
		},
		{
			name:       "NullInt64 to int32",
			nullable:   &sql.NullInt64{Int64: 1001, Valid: true},
			targetType: reflect.TypeFor[int32](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, int32(1001), int32(v.Int()))
			},
		},
		{
			name:       "NullInt64 to uint8",
			nullable:   &sql.NullInt64{Int64: 255, Valid: true},
			targetType: reflect.TypeFor[uint8](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, uint8(255), uint8(v.Uint()))
			},
		},

		// --- NullFloat64 ---
		{
			name:       "NullFloat64 to float64",
			nullable:   &sql.NullFloat64{Float64: 3.25, Valid: true},
			targetType: reflect.TypeFor[float64](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.InEpsilon(t, 3.25, v.Float(), 1e-9)
			},
		},
		{
			name:       "NullFloat64 to *float64",
			nullable:   &sql.NullFloat64{Float64: -1.5, Valid: true},
			targetType: reflect.TypeFor[*float64](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.InEpsilon(t, -1.5, v.Elem().Float(), 1e-9)
			},
		},

		// --- NullBool ---
		{
			name:       "NullBool to bool true",
			nullable:   &sql.NullBool{Bool: true, Valid: true},
			targetType: reflect.TypeFor[bool](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.True(t, v.Bool())
			},
		},
		{
			name:       "NullBool to *bool false",
			nullable:   &sql.NullBool{Bool: false, Valid: true},
			targetType: reflect.TypeFor[*bool](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.False(t, v.Elem().Bool())
			},
		},

		// --- NullTime ---
		{
			name:       "NullTime to time.Time",
			nullable:   &sql.NullTime{Time: fixTime, Valid: true},
			targetType: reflect.TypeFor[time.Time](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.True(t, fixTime.Equal(v.Interface().(time.Time)))
			},
		},
		{
			name:       "NullTime to *time.Time",
			nullable:   &sql.NullTime{Time: fixTime, Valid: true},
			targetType: reflect.TypeFor[*time.Time](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				got := v.Interface().(*time.Time)
				require.True(t, fixTime.Equal(*got))
			},
		},

		// --- default branch: non-nullable wrapped value (e.g. direct scan slot) ---
		{
			name:       "[]byte to string",
			nullable:   zutil.Ptr([]byte("bytes")),
			targetType: reflect.TypeFor[string](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, "bytes", v.String())
			},
		},
		{
			name:       "string to string default path",
			nullable:   zutil.Ptr("direct"),
			targetType: reflect.TypeFor[string](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, "direct", v.String())
			},
		},
		{
			name:       "int to *int default path",
			nullable:   zutil.Ptr(99),
			targetType: reflect.TypeFor[*int](),
			wantOK:     true,
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, 99, int(v.Elem().Int()))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := convertNullableValue(tc.nullable, tc.targetType)
			require.Equal(t, tc.wantOK, ok, "convertNullableValue ok flag")
			if !tc.wantOK {
				want := reflect.Zero(tc.targetType)
				require.True(t, got.IsValid())
				require.Equal(t, want.Interface(), got.Interface(), "expected zero value of target type")
				return
			}
			require.True(t, got.IsValid(), "result must be valid when ok")
			require.NotNil(t, tc.check)
			tc.check(t, got)
		})
	}
}

// nullableRoundtripStruct holds one field per branch of createNullableScanTarget (helpers.go),
// matching how relation JOIN columns allocate scan slots (sql.Null* wrappers).
type nullableRoundtripStruct struct {
	S    string
	PStr *string

	I64  int64
	PI64 *int64
	U8   uint8

	F64  float64
	PF64 *float64

	B  bool
	PB *bool

	T  time.Time
	PT *time.Time

	PH *ptrScanReceiver
	HV ptrScanReceiver
}

func TestCreateNullableScanTargetConvertRoundtrip(t *testing.T) {
	t.Parallel()

	fixTime := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
	heapStr := "from-heap"
	heapTime := fixTime

	typ := reflect.TypeFor[nullableRoundtripStruct]()

	cases := []struct {
		name    string
		field   string
		setup   func(t *testing.T, slot interface{})
		check   func(t *testing.T, v reflect.Value)
		invalid bool // expect convertNullableValue to report !ok (e.g. SQL NULL)
	}{
		{
			field: "S",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				ns := slot.(*sql.NullString)
				ns.String, ns.Valid = "plain", true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, "plain", v.String())
			},
		},
		{
			field: "S",
			name:  "S_NULL",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				ns := slot.(*sql.NullString)
				ns.Valid = false
			},
			check:   func(t *testing.T, v reflect.Value) {},
			invalid: true,
		},
		{
			field: "PStr",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				ns := slot.(*sql.NullString)
				ns.String, ns.Valid = heapStr, true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, heapStr, v.Elem().String())
			},
		},
		{
			field: "I64",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				n := slot.(*sql.NullInt64)
				n.Int64, n.Valid = -42, true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, int64(-42), v.Int())
			},
		},
		{
			field: "PI64",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				n := slot.(*sql.NullInt64)
				n.Int64, n.Valid = 99, true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, int64(99), v.Elem().Int())
			},
		},
		{
			field: "U8",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				n := slot.(*sql.NullInt64)
				n.Int64, n.Valid = 201, true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, uint8(201), uint8(v.Uint()))
			},
		},
		{
			field: "F64",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				n := slot.(*sql.NullFloat64)
				n.Float64, n.Valid = 2.5, true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.InEpsilon(t, 2.5, v.Float(), 1e-9)
			},
		},
		{
			field: "PF64",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				n := slot.(*sql.NullFloat64)
				n.Float64, n.Valid = -0.25, true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.InEpsilon(t, -0.25, v.Elem().Float(), 1e-9)
			},
		},
		{
			field: "B",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				n := slot.(*sql.NullBool)
				n.Bool, n.Valid = true, true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.True(t, v.Bool())
			},
		},
		{
			field: "PB",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				n := slot.(*sql.NullBool)
				n.Bool, n.Valid = false, true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.False(t, v.Elem().Bool())
			},
		},
		{
			field: "T",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				n := slot.(*sql.NullTime)
				n.Time, n.Valid = fixTime, true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.True(t, fixTime.Equal(v.Interface().(time.Time)))
			},
		},
		{
			field: "PT",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				n := slot.(*sql.NullTime)
				n.Time, n.Valid = heapTime, true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				got := v.Interface().(*time.Time)
				require.True(t, heapTime.Equal(*got))
			},
		},
		{
			field: "PH",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				ns := slot.(*sql.NullString)
				ns.String, ns.Valid = `{"payload":"json-col"}`, true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				got := v.Interface().(*ptrScanReceiver)
				require.Equal(t, `{"payload":"json-col"}`, got.Payload)
			},
		},
		{
			field: "HV",
			setup: func(t *testing.T, slot interface{}) {
				t.Helper()
				ns := slot.(*sql.NullString)
				ns.String, ns.Valid = "value-field", true
			},
			check: func(t *testing.T, v reflect.Value) {
				t.Helper()
				require.Equal(t, "value-field", v.FieldByName("Payload").String())
			},
		},
	}

	for _, tc := range cases {
		sub := tc.name
		if sub == "" {
			sub = tc.field
		}
		t.Run(sub, func(t *testing.T) {
			t.Parallel()
			sf, ok := typ.FieldByName(tc.field)
			require.True(t, ok, "field %q on nullableRoundtripStruct", tc.field)

			slot := createNullableScanTarget(sf.Type)
			require.NotNil(t, slot)
			tc.setup(t, slot)

			wantOK := !tc.invalid
			got, convOK := convertNullableValue(slot, sf.Type)
			require.Equal(t, wantOK, convOK, "convertNullableValue after createNullableScanTarget")

			if !wantOK {
				want := reflect.Zero(sf.Type)
				require.True(t, got.IsValid())
				require.Equal(t, want.Interface(), got.Interface())
				return
			}
			tc.check(t, got)
		})
	}
}
