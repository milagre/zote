package zcmd

import (
	"fmt"

	"github.com/spf13/cast"
	"github.com/urfave/cli/v2"
)

type Env struct {
	context *cli.Context
}

func (e Env) CommandName() string {
	return e.context.Command.Name
}

func (e Env) Bool(name string) bool {
	f := e.flag(name)

	if _, ok := f.(*cli.BoolFlag); ok {
		if e.context.IsSet(name) {
			return true
		}
	}

	return false
}

func (e Env) Int(name string) int {
	v := e.context.Value(name)

	i, err := cast.ToIntE(v)
	if err == nil {
		return i
	}

	return 0
}

func (e Env) String(name string) string {
	v := e.context.Value(name)
	f := e.flag(name)

	if _, ok := f.(*cli.StringFlag); ok {
		s, ok := v.(string)
		if ok {
			return s
		}
	}

	return fmt.Sprintf("%s", v)
}

func (e Env) flag(name string) cli.Flag {
	for _, c := range e.context.Lineage() {
		if c.Command == nil {
			continue
		}

		for _, f := range c.Command.Flags {
			for _, n := range f.Names() {
				if n == name {
					return f
				}
			}
		}
	}

	if e.context.App != nil {
		for _, f := range e.context.App.Flags {
			for _, n := range f.Names() {
				if n == name {
					return f
				}
			}
		}
	}

	return nil
}
