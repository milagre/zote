package zcmd

import (
	"context"
)

type Aspect interface {
	Apply(r Configurable)
}

type Configurable interface {
	AddBool(name string) BoolFlag
	AddInt(name string) IntFlag
	AddString(name string) StringFlag
}

type RunFunc func(ctx context.Context, env Env) error
