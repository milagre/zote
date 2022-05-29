package sort

import (
	"github.com/milagre/zote/go/element/elem"
)

type Direction int8

const (
	Asc  Direction = iota
	Desc Direction = iota
)

type Sort struct {
	Field     elem.Field
	Direction Direction
}
