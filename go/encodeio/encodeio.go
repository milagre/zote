package encodeio

import (
	"fmt"
	"io"
)

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
