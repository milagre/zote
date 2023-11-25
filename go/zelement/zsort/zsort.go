package zsort

import (
	"github.com/milagre/zote/go/zelement/zelem"
)

type Direction int8

const (
	Asc  Direction = iota
	Desc Direction = iota
)

type Sort struct {
	Field     zelem.Field
	Direction Direction
}
