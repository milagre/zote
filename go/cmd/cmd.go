package cmd

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/urfave/cli/v2"

	zotelog "github.com/milagre/zote/go/log"
)

type App struct {
	app       *cli.App
	envPrefix string
}

type Aspect interface {
	Apply(r Configurable)
}

type Configurable interface {
	AddBool(name string) BoolFlag
	AddInt(name string) IntFlag
	AddString(name string) StringFlag
}

type Command struct {
	Config  Aspect
	Run     RunFunc
	command *cli.Command
	app     *App
}

type RunFunc func(ctx context.Context, env Env) error

func NewApp(
	name string,
	envPrefix string,
	config Aspect,
	commands map[string]Command,
) App {
	app := App{
		envPrefix: envPrefix,
		app: &cli.App{
			Name:     name,
			Commands: make([]*cli.Command, 0, len(commands)),
		},
	}

	for name, command := range commands {
		command.command = newCmd(name, command)
		command.app = &app
		if command.Config != nil {
			command.Config.Apply(command)
		}
		app.app.Commands = append(
			app.app.Commands,
			command.command,
		)
	}

	if config != nil {
		config.Apply(&app)
	}

	return app
}

func (a App) Run(ctx context.Context) {
	a.RunArgs(ctx, os.Args)
}

func (a App) RunArgs(ctx context.Context, args []string) {
	for _, c := range a.app.Commands {
		c.Flags = append(c.Flags, a.app.Flags...)
	}
	a.app.Flags = []cli.Flag{}

	home, err := os.UserHomeDir()
	if err != nil {
		zotelog.FromContext(ctx).
			WithField("error", err).
			Warnf("Finding user home dir for configuration file.")
	} else {
		cfg := home + string(os.PathSeparator) + "." + a.app.Name
		err := LoadFile(cfg, a.envPrefix)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				zotelog.FromContext(ctx).
					Debugf("Configuration file not found in user home dir.")
			} else {
				zotelog.FromContext(ctx).
					WithField("error", err).
					Warnf("Reading configuration file at ")
			}
		}
	}

	err = a.app.RunContext(ctx, args)
	if err != nil {
		zotelog.FromContext(ctx).
			WithField("error", err).
			Errorf("Application terminated with error.")
		os.Exit(1)
	}
}

func (a *App) AddBool(name string) BoolFlag {
	f := newBoolFlag(a, name)
	a.app.Flags = append(a.app.Flags, f.flag)
	return f
}

func (a *App) AddInt(name string) IntFlag {
	f := newIntFlag(a, name)
	a.app.Flags = append(a.app.Flags, f.flag)
	return f
}

func (a *App) AddString(name string) StringFlag {
	f := newStringFlag(a, name)
	a.app.Flags = append(a.app.Flags, f.flag)
	return f
}

func (a App) envVar(name string) string {
	return strcase.ToScreamingSnake(
		strings.Join(
			[]string{strings.Trim(a.envPrefix, "_"), name},
			"_",
		),
	)
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
