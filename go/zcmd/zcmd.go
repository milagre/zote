// Package zcmd provides a CLI application framework with environment-based
// configuration, built on urfave/cli.
//
// # Application Setup
//
//	app := zcmd.NewApp(
//		"myapp",         // Application name
//		"MA",            // Environment variable prefix
//		globalConfig,    // Global Aspect for configuration
//		map[string]zcmd.Command{
//			"serve":   {Config: serveConfig, Run: serveRun},
//			"migrate": {Config: migrateConfig, Run: migrateRun},
//		},
//	)
//	app.Run(ctx)
//
// # Aspect Configuration
//
// Aspects enable modular configuration setup through the Configurable interface:
//
//	func Config(c zcmd.Configurable) {
//		c.AddString("hostname").Required()
//		c.AddInt("port").Default(8080)
//		c.AddBool("debug")
//	}
//
// # Environment Variables
//
// Flags are automatically mapped to environment variables using the prefix:
//
//	Flag "database-url" with prefix "MA" → MA_DATABASE_URL
//
// # Configuration File
//
// The framework loads environment variables from ~/.<appname> (e.g., ~/.myapp)
// in .env format via joho/godotenv.
package zcmd

import (
	"context"
)

type Aspect interface {
	Apply(c Configurable)
}

type Configurable interface {
	AddBool(name string) BoolFlag
	AddInt(name string) IntFlag
	AddString(name string) StringFlag
}

type RunFunc func(ctx context.Context, env Env) error
