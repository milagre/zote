package zencodeio_test

import (
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/milagre/zote/go/zencodeio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for JSON encoding/decoding
type testStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// TestJSONEncoder tests the JSON encoder functionality
func TestJSON(t *testing.T) {
	data := testStruct{Name: "test", Value: 42}
	var decoded testStruct
	err := zencodeio.Read(
		zencodeio.NewJSONEncoder(data),
		zencodeio.NewJSONDecoder(&decoded),
	)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

// TestJSONDecoder tests the JSON decoder functionality
func TestJSONDecoder(t *testing.T) {
	t.Run("decode simple struct", func(t *testing.T) {
		expected := testStruct{Name: "test", Value: 42}
		jsonData, err := json.Marshal(expected)
		require.NoError(t, err)

		var result testStruct
		decoder := zencodeio.NewJSONDecoder(&result)

		n, err := decoder.Write(jsonData)
		require.NoError(t, err)
		assert.Equal(t, len(jsonData), n)

		err = decoder.Close()
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("write after close returns error", func(t *testing.T) {
		expected := testStruct{Name: "test", Value: 42}
		jsonData, err := json.Marshal(expected)
		require.NoError(t, err)

		var result testStruct
		decoder := zencodeio.NewJSONDecoder(&result)

		_, err = decoder.Write(jsonData)
		require.NoError(t, err)

		err = decoder.Close()
		require.NoError(t, err)

		// Try to write after close
		_, err = decoder.Write([]byte("more data"))
		assert.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("invalid json returns error on close", func(t *testing.T) {
		var result testStruct
		decoder := zencodeio.NewJSONDecoder(&result)

		_, err := decoder.Write([]byte("invalid json{"))
		require.NoError(t, err)

		err = decoder.Close()
		assert.Error(t, err)
	})
}

// TestMarshallerEncoder tests the generic marshaller encoder
func TestMarshallerEncoder(t *testing.T) {
	t.Run("custom marshal function", func(t *testing.T) {
		data := "custom data"
		marshalFunc := func(v any) ([]byte, error) {
			s, ok := v.(string)
			if !ok {
				return nil, errors.New("not a string")
			}
			return []byte("CUSTOM:" + s), nil
		}

		encoder := zencodeio.NewMarshallerEncoder(data, marshalFunc)

		var result string
		decoder := zencodeio.NewMarshallerDecoder(&result, "custom/type", func(data []byte, v any) error {
			ptr := v.(*string)
			*ptr = string(data)
			return nil
		})

		err := zencodeio.Read(encoder, decoder)
		require.NoError(t, err)
		assert.Equal(t, "CUSTOM:custom data", result)
	})

	t.Run("marshal error propagates", func(t *testing.T) {
		data := "test"
		marshalErr := errors.New("marshal failed")
		marshalFunc := func(v any) ([]byte, error) {
			return nil, marshalErr
		}

		encoder := zencodeio.NewMarshallerEncoder(data, marshalFunc)

		var result string
		decoder := zencodeio.NewMarshallerDecoder(&result, "custom/type", func(data []byte, v any) error {
			ptr := v.(*string)
			*ptr = string(data)
			return nil
		})

		err := zencodeio.Read(encoder, decoder)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "marshaling")
	})
}

// TestMarshallerDecoder tests the generic marshaller decoder
func TestMarshallerDecoder(t *testing.T) {
	t.Run("custom unmarshal function", func(t *testing.T) {
		unmarshalFunc := func(data []byte, v any) error {
			ptr, ok := v.(*string)
			if !ok {
				return errors.New("not a string pointer")
			}
			*ptr = "UNMARSHALED:" + string(data)
			return nil
		}

		var result string
		decoder := zencodeio.NewMarshallerDecoder(&result, "custom/type", unmarshalFunc)

		_, err := decoder.Write([]byte("input"))
		require.NoError(t, err)

		err = decoder.Close()
		require.NoError(t, err)
		assert.Equal(t, "UNMARSHALED:input", result)
	})

	t.Run("unmarshal error propagates", func(t *testing.T) {
		unmarshalErr := errors.New("unmarshal failed")
		unmarshalFunc := func(data []byte, v any) error {
			return unmarshalErr
		}

		var result string
		decoder := zencodeio.NewMarshallerDecoder(&result, "custom/type", unmarshalFunc)

		_, err := decoder.Write([]byte("input"))
		require.NoError(t, err)

		err = decoder.Close()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshaling")
	})

	t.Run("accumulates multiple writes", func(t *testing.T) {
		var accumulated []byte
		unmarshalFunc := func(data []byte, v any) error {
			accumulated = data
			return nil
		}

		var result string
		decoder := zencodeio.NewMarshallerDecoder(&result, "custom/type", unmarshalFunc)

		_, err := decoder.Write([]byte("part1"))
		require.NoError(t, err)

		_, err = decoder.Write([]byte("part2"))
		require.NoError(t, err)

		_, err = decoder.Write([]byte("part3"))
		require.NoError(t, err)

		err = decoder.Close()
		require.NoError(t, err)

		assert.Equal(t, "part1part2part3", string(accumulated))
	})
}

// TestEncoderDecoderRoundTrip tests full round-trip encoding and decoding
func TestEncoderDecoderRoundTrip(t *testing.T) {
	t.Run("json round-trip with various types", func(t *testing.T) {
		testCases := []struct {
			name string
			data interface{}
		}{
			{
				name: "string",
				data: "hello world",
			},
			{
				name: "int",
				data: 42,
			},
			{
				name: "float",
				data: 3.14159,
			},
			{
				name: "bool",
				data: true,
			},
			{
				name: "slice",
				data: []string{"a", "b", "c"},
			},
			{
				name: "map",
				data: map[string]int{"one": 1, "two": 2},
			},
			{
				name: "struct",
				data: testStruct{Name: "test", Value: 123},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Decode into any for comparison
				var result any
				err := zencodeio.Read(
					zencodeio.NewJSONEncoder(tc.data),
					zencodeio.NewJSONDecoder(&result),
				)
				require.NoError(t, err)

				// Re-encode both original and decoded to compare JSON representations
				expectedJSON, err := json.Marshal(tc.data)
				require.NoError(t, err)

				resultJSON, err := json.Marshal(result)
				require.NoError(t, err)

				assert.JSONEq(t, string(expectedJSON), string(resultJSON))
			})
		}
	})
}
