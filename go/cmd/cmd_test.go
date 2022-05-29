package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	zotelog "github.com/milagre/zote/go/log"
	zotelogrus "github.com/milagre/zote/go/log/logrus"
)

type stringsAspectType struct{}

func (a stringsAspectType) Apply(c Configurable) {
	c.AddString("str")
	c.AddString("str-default").Default("default")
}

type intsAspectType struct{}

func (a intsAspectType) Apply(c Configurable) {
	c.AddInt("int")
	c.AddInt("int-default").Default(5)
}

type boolsAspectType struct{}

func (a boolsAspectType) Apply(c Configurable) {
	c.AddBool("bool")
}

var stringsAspect stringsAspectType
var intsAspect intsAspectType
var boolsAspect boolsAspectType

func makeApp(cb func(e Env)) App {
	var runme = func(_ context.Context, e Env) error { cb(e); return nil }
	app := NewApp(
		"runme",
		"ZOTE",
		nil,
		map[string]Command{
			"strings": {Run: runme, Config: stringsAspect},
			"ints":    {Run: runme, Config: intsAspect},
			"bools":   {Run: runme, Config: boolsAspect},
		},
	)

	return app
}

func runTest(t *testing.T, wantCrash bool, command string, args []string, env []string, test func(e Env)) {
	t.Helper()

	if os.Getenv("ZOTE_TEST_EXECUTOR") == "1" {
		ctx := zotelog.Context(
			context.Background(),
			zotelog.New(zotelog.LevelInfo, zotelogrus.New(zotelog.LevelInfo)),
		)

		executed := false
		app := makeApp(func(e Env) {
			executed = true
			test(e)
		})

		app.RunArgs(ctx, append(
			[]string{
				"runme",
				command,
			},
			args...,
		))

		require.True(t, executed, "command was not executed")
		return
	}

	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	parts := strings.Split(frame.Function, ".")
	fn := parts[len(parts)-1]

	cmd := exec.Command(os.Args[0], fmt.Sprintf("-test.run=^%s$", fn))
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "ZOTE_TEST_EXECUTOR=1")
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok && !e.Success() && !wantCrash {
			t.Fatalf("command expected to succeed, but didn't - %s: %s\n%s", e.Error(), string(out), string(e.Stderr))
		} else if !wantCrash {
			t.Fatalf("command failed in an unexpected fashion - %s: %s", err.Error(), string(out))
		}
	} else if wantCrash {
		t.Fatalf("command exited successfully unexpectedly")
	}
}

func TestStringsFlags(t *testing.T) {
	runTest(t, false, "strings",
		[]string{"--str=one", "--str-default=two"},
		[]string{},
		func(e Env) {
			assert.Equal(t, "one", e.String("str"))
			assert.Equal(t, "two", e.String("str-default"))
		},
	)
}

func TestStringsEnv(t *testing.T) {
	runTest(t, false, "strings",
		[]string{},
		[]string{"ZOTE_STR=one", "ZOTE_STR_DEFAULT=two"},
		func(e Env) {
			assert.Equal(t, "one", e.String("str"))
			assert.Equal(t, "two", e.String("str-default"))
		},
	)
}

func TestStringsDefault(t *testing.T) {
	runTest(t, false, "strings",
		[]string{"--str=one"},
		[]string{},
		func(e Env) {
			assert.Equal(t, "default", e.String("str-default"))
		},
	)
}

func TestStringsMissingRequired(t *testing.T) {
	runTest(t, true, "strings",
		[]string{},
		[]string{},
		func(e Env) {},
	)
}

func TestIntsFlags(t *testing.T) {
	runTest(t, false, "ints",
		[]string{"--int=1", "--int-default=2"},
		[]string{},
		func(e Env) {
			assert.Equal(t, 1, e.Int("int"))
			assert.Equal(t, 2, e.Int("int-default"))
		},
	)
}

func TestIntsEnv(t *testing.T) {
	runTest(t, false, "ints",
		[]string{},
		[]string{"ZOTE_INT=1", "ZOTE_INT_DEFAULT=2"},
		func(e Env) {
			assert.Equal(t, 1, e.Int("int"))
			assert.Equal(t, 2, e.Int("int-default"))
		},
	)
}

func TestIntsDefault(t *testing.T) {
	runTest(t, false, "ints",
		[]string{"--int=1"},
		[]string{},
		func(e Env) {
			assert.Equal(t, 5, e.Int("int-default"))
		},
	)
}

func TestIntsMissingRequired(t *testing.T) {
	runTest(t, true, "ints",
		[]string{},
		[]string{},
		func(e Env) {},
	)
}
