package zencodeio

import (
	"fmt"
	"io"
)

type MarshalFunc func(interface{}) ([]byte, error)

type UnmarshalFunc func([]byte, interface{}) error

type MarshallerEncoder struct {
	v       interface{}
	marshal MarshalFunc

	buf []byte
}

func NewMarshallerEncoder(v interface{}, marshal MarshalFunc) Encoder {
	return &MarshallerEncoder{v: v, marshal: marshal}
}

func (e *MarshallerEncoder) Read(p []byte) (int, error) {
	if e.v == nil {
		n := copy(p, e.buf)
		e.buf = e.buf[n:]

		var err error
		if len(e.buf) == 0 {
			err = io.EOF
		}

		return n, err
	}

	data, err := e.marshal(e.v)
	if err != nil {
		return 0, fmt.Errorf("marshaling: %w", err)
	}

	e.buf = data
	e.v = nil

	return e.Read(p)
}

type MarshallerDecoder struct {
	v         interface{}
	encoding  string
	unmarshal UnmarshalFunc

	closed bool
	buf    []byte
}

func NewMarshallerDecoder(v interface{}, encoding string, unmarshal UnmarshalFunc) Decoder {
	return &MarshallerDecoder{v: v, unmarshal: unmarshal, encoding: encoding}
}

func (e *MarshallerDecoder) Write(p []byte) (n int, err error) {
	if e.closed {
		return 0, io.ErrClosedPipe
	}

	e.buf = append(e.buf, p...)

	return len(p), nil
}

func (e *MarshallerDecoder) Close() error {
	if e.closed {
		return nil
	}

	err := e.unmarshal(e.buf, e.v)
	if err != nil {
		return fmt.Errorf("unmarshaling: %w", err)
	}

	e.closed = true

	return nil
}

func (e *MarshallerDecoder) Encoding() string {
	return e.encoding
}
