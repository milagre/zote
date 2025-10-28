# zcmd - CLI Application Framework

CLI application framework with environment-based configuration.

Build on urfave/cli under the hood.

## Usage

### Basic Application Setup

```go
app := zcmd.NewApp(
    "myapp",                     // Application name
    "MA",                        // Environment variable prefix
    globalConfig,                // Global aspect for configuration
    map[string]zcmd.Command{
        "serve": {
            Config: serveConfig, // Command-specific aspect for configuration
            Run: serveRun        // Entry-point for command
        }, 
        "migrate": {Config: migrateConfig, Run: migrateRun},
    },
)
app.Run(ctx)
```

### Aspect Configuration

Aspects allow modular configuration setup:

```go
type Aspect interface {
    Apply(c Configurable)
}

// Usage in command config
func Config(c zcmd.Configurable) {
    c.AddString("hostname").Required()
    c.AddInt("port").Default(8080)
    c.AddBool("debug")
}
```

### Environment Variables

Flags are automatically mapped to environment variables:
- Flag `database-url` with prefix `MA` becomes `MA_DATABASE_URL`

### Configuration File

The framework automatically looks for a configuration file in the user's home directory:
- Location: `~/.<appname>` (e.g., `~/.myapp`)
- Format: `.env` file format via `joho/godotenv`
- Environment variables from the file are loaded with the specified prefix

### Terraform Modules
This framework is designed to enable configuration of installations into kubernetes directly through the terraform modules in `terraform/k8s`