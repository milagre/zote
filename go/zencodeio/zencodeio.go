// Package zencodeio provides stream-based marshalling and unmarshalling for Go
// data structures without inline error handling.
//
// zencodeio abstracts un/marshalling data structures directly to/from streams,
// eliminating manual read/write and marshal/unmarshal steps.
//
// # Encoding
//
// Encoders implement io.Reader, producing marshalled data on read:
//
//	data := MyStruct{Name: "test", Value: 42}
//	io.Copy(w, zencodeio.NewJSONEncoder(data))
//
// # Decoding
//
// Decoders implement io.WriteCloser, unmarshalling written data into a target.
// zencodeio.Read consumes the reader and closes the decoder. Unmarshalling is
// performed on close.
//
//	var result MyStruct
//	err := zencodeio.Read(r, zencodeio.NewJSONDecoder(&result))
//
// # Custom Formats
//
// Use NewMarshallerEncoder and NewMarshallerDecoder for custom serialization:
//
//	encoder := zencodeio.NewMarshallerEncoder(data, func(v any) ([]byte, error) {
//	    return []byte(fmt.Sprintf("CUSTOM:%v", v)), nil
//	})
//	io.Copy(writer, encoder)
package zencodeio

import (
	"fmt"
	"io"
)

type Encoder interface {
	io.Reader
}

type Decoder interface {
	io.WriteCloser
	Encoding() string
}

func Read(r io.Reader, d Decoder) error {
	if _, err := io.ReadAll(io.TeeReader(r, d)); err != nil {
		return fmt.Errorf("reading into decoder: %w", err)
	}

	if err := d.Close(); err != nil {
		return fmt.Errorf("closing decoder: %w", err)
	}

	return nil
}
