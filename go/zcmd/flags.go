package zcmd

import "github.com/urfave/cli/v2"

// BoolFlag
type BoolFlag struct {
	flag *cli.BoolFlag
}

func newBoolFlag(a *App, name string) BoolFlag {
	return BoolFlag{
		flag: &cli.BoolFlag{
			Name:     name,
			Required: false,
			EnvVars:  []string{a.envVar(name)},
		},
	}
}

// IntFlag
type IntFlag struct {
	flag *cli.IntFlag
}

func newIntFlag(a *App, name string) IntFlag {
	return IntFlag{
		flag: &cli.IntFlag{
			Name:     name,
			Required: true,
			EnvVars:  []string{a.envVar(name)},
		},
	}
}

func (f IntFlag) Default(i int) {
	f.flag.Value = i
	f.flag.Required = false
}

// StringFlag
type StringFlag struct {
	flag *cli.StringFlag
}

func newStringFlag(a *App, name string) StringFlag {
	return StringFlag{
		flag: &cli.StringFlag{
			Name:     name,
			Required: true,
			EnvVars:  []string{a.envVar(name)},
		},
	}
}

func (f StringFlag) Default(s string) {
	f.flag.Value = s
	f.flag.Required = false
}
