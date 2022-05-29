package encodeio

import (
	"encoding/json"
	"fmt"
	"io"
)

type JSONEncoder struct {
	v   interface{}
	buf []byte
}

func NewJSONEncoder(v interface{}) io.Reader {
	return &JSONEncoder{v: v}
}

func (e *JSONEncoder) Read(p []byte) (int, error) {
	if e.v == nil {
		n := copy(p, e.buf)
		e.buf = e.buf[n:]

		var err error
		if len(e.buf) == 0 {
			err = io.EOF
		}

		return n, err
	}

	data, err := json.Marshal(e.v)
	if err != nil {
		return 0, fmt.Errorf("marshaling json: %w", err)
	}

	e.buf = data
	e.v = nil

	return e.Read(p)
}

type JSONDecoder struct {
	v      interface{}
	closed bool
	buf    []byte
}

func NewJSONDecoder(v interface{}) Decoder {
	return &JSONDecoder{v: v}
}

func (e *JSONDecoder) Write(p []byte) (n int, err error) {
	if e.closed {
		return 0, io.ErrClosedPipe
	}

	e.buf = append(e.buf, p...)

	return len(p), nil
}

func (e *JSONDecoder) Close() error {
	if e.closed {
		return nil
	}

	err := json.Unmarshal(e.buf, e.v)
	if err != nil {
		return fmt.Errorf("unmarshaling json: %w", err)
	}

	e.closed = true

	return nil
}

func (e *JSONDecoder) Encoding() string {
	return "application/json"
}
