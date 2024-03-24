package zsort

import "github.com/milagre/zote/go/zelement"

type Direction int8

const (
	Asc  Direction = iota
	Desc Direction = iota
)

type Sort struct {
	Element   zelement.Element
	Direction Direction
}
