package zcmd

import (
	"github.com/urfave/cli/v2"
)

type Command struct {
	Config  Aspect
	Run     RunFunc
	command *cli.Command
	app     *App
}

func newCmd(name string, cmd Command) *cli.Command {
	return &cli.Command{
		Name: name,
		Action: func(c *cli.Context) error {
			return cmd.Run(
				c.Context,
				Env{context: c},
			)
		},
	}
}

func (c Command) AddBool(name string) BoolFlag {
	f := newBoolFlag(c.app, name)
	c.command.Flags = append(c.command.Flags, f.flag)
	return f
}

func (c Command) AddInt(name string) IntFlag {
	f := newIntFlag(c.app, name)
	c.command.Flags = append(c.command.Flags, f.flag)
	return f
}

func (c Command) AddString(name string) StringFlag {
	f := newStringFlag(c.app, name)
	c.command.Flags = append(c.command.Flags, f.flag)
	return f
}
