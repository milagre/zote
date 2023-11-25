package zcmd

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/milagre/zote/go/zbuild"
	zotelog "github.com/milagre/zote/go/zlog"
	"github.com/urfave/cli/v2"
)

type App struct {
	app       *cli.App
	envPrefix string
}

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
			Version:  zbuild.Version(),
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
