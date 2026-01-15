# CLI - Cobra Command Framework with Viper Configuration

A lightweight Go library that simplifies building CLI applications with [Cobra](https://github.com/spf13/cobra) commands and [Viper](https://github.com/spf13/viper) configuration management. It provides sensible defaults for configuration loading, structured logging, and type-safe command execution.

## Features

- **Type-safe command builders** using Go generics for zero-boilerplate command setup
- **Automatic configuration loading** from files, environment variables, and flags
- **Structured logging** with [zerolog](https://github.com/rs/zerolog) and file rotation support
- **Context management** for passing application metadata through command execution
- **Flexible command options** for customizing command behavior

## Installation

```bash
go get github.com/dioad/cli
```

## Quick Start

### Basic Command with Configuration

```go
package main

import (
	"context"
	"fmt"
	"github.com/dioad/cli"
	"github.com/spf13/cobra"
)

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

func main() {
	cfg := &AppConfig{
		Name:    "myapp",
		Version: "1.0.0",
	}

	cmd := &cobra.Command{
		Use:   "greet",
		Short: "Greet the user",
	}

	// Register command with type-safe config handling
	cli.NewCommand(cmd, greetCommand, cfg)

	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}

func greetCommand(ctx context.Context, cfg *AppConfig) error {
	fmt.Printf("Hello from %s v%s\n", cfg.Name, cfg.Version)
	return nil
}
```

### With Configuration File

Configuration files are loaded from multiple locations in order of precedence:

1. Path specified by `--config` flag
2. Environment variables with prefix `APPNAME_`
3. User config directory: `$HOME/.config/{orgName}/{appName}/config.yaml`
4. System config: `/etc/{orgName}/{appName}/config.yaml`
5. Current directory: `./config.yaml`

Example `config.yaml`:

```yaml
name: myapp
version: 1.0.0
log:
  level: debug
  file: "~/.myapp/logs/app.log"
  max-size: 100    # MB
  max-age: 7       # days
  max-backups: 3
  compress: true
```

## Core Components

### Command Builder

Create Cobra commands with type-safe configuration:

```go
func NewCommand[T any](
	cmd *cobra.Command,
	runFunc func(context.Context, *T) error,
	defaultConfig *T,
	opts ...CommandOpt,
) *cobra.Command
```

**Example:**

```go
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

cfg := &ServerConfig{Port: 8080, Host: "localhost"}

cmd := cli.NewCommand(
	&cobra.Command{
		Use:   "serve",
		Short: "Start the server",
	},
	func(ctx context.Context, cfg *ServerConfig) error {
		fmt.Printf("Starting server on %s:%d\n", cfg.Host, cfg.Port)
		return nil
	},
	cfg,
	cli.WithConfigFlag("config.yaml"),
)
```

### Configuration Management

#### InitViperConfig

Initialize Viper with flag parsing and config loading:

```go
type Config struct {
	Debug   bool   `mapstructure:"debug"`
	LogFile string `mapstructure:"log-file"`
}

cfg := &Config{}
err := cli.InitViperConfig("myorg", "myapp", cfg)
if err != nil {
	panic(err)
}
```

#### InitConfig

Load configuration from files and environment with Cobra command context:

```go
func InitConfig(
	orgName, appName string,
	cmd *cobra.Command,
	cfgFile string,
	cfg interface{},
) (*CommonConfig, error)
```

Returns `CommonConfig` with logging configuration and populates the provided config struct.

### Path Helpers

Get default paths for configuration and persistence:

```go
// Get user's config directory
configPath, err := cli.DefaultConfigPath("myorg", "myapp")

// Get persistence directory (uses /persist in Docker)
persistPath, err := cli.DefaultPersistencePath("myorg", "myapp")

// Get full config file path
configFile, err := cli.DefaultConfigFile("myorg", "myapp", "config")
```

### Context Management

Store and retrieve application metadata through context:

```go
ctx := cli.Context(
	context.Background(),
	cli.SetOrgName("myorg"),
	cli.SetAppName("myapp"),
)

// Later, retrieve from context
orgName := cli.getOrgName(ctx)
appName := cli.getAppName(ctx)
```

### Logging

The `logging` subpackage provides structured logging via zerolog:

```go
import "github.com/dioad/cli/logging"
import "github.com/rs/zerolog/log"

// Configure logging from config
loggingCfg := logging.Config{
	Level:      "debug",
	File:       "~/.myapp/logs/app.log",
	MaxSize:    100,
	MaxBackups: 3,
	Compress:   true,
}

logging.ConfigureCmdLogger(loggingCfg)

// Use structured logging
log.Info().
	Str("user", "alice").
	Int("count", 42).
	Msg("operation completed")
```

**Log Levels:** `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`

## Architecture Overview

```
Package cli
├── Command Building (NewCommand, CommandOpt)
├── Config Management (InitViperConfig, InitConfig)
├── Path Helpers (DefaultConfigPath, DefaultPersistencePath)
├── Context Management (SetOrgName, SetAppName)
└── Cobra Integration (CobraRunE, CobraRunEWithConfig)

Package logging
├── Configuration (Config struct)
├── Level Management (ConfigureLogLevel)
├── Output Management (ConfigureLogOutput, ConfigureLogFileOutput)
└── Logging Options (WithDefaultLogLevel)
```

## Common Patterns

### Multi-Level Commands

```go
rootCmd := &cobra.Command{Use: "app"}
ctx := cli.Context(
	context.Background(),
	cli.SetOrgName("myorg"),
	cli.SetAppName("myapp"),
)
rootCmd.SetContext(ctx)

// Add subcommands
rootCmd.AddCommand(
	cli.NewCommand(
		&cobra.Command{Use: "start"},
		startCommand,
		&ServerConfig{},
		cli.WithConfigFlag("config.yaml"),
	),
)
```

### Environment Variable Override

All configuration values can be overridden via environment variables using the format:

```
{APPNAME}_{SECTION}_{FIELD}
```

Example:
```bash
MYAPP_LOG_LEVEL=debug MYAPP_PORT=9000 ./myapp
```

### Docker Support

The library automatically detects Docker environments:

```go
if cli.IsDocker() {
	// Use /persist and /config directories
}
```

Configuration and persistence paths default to container volumes when running in Docker.

## Examples

See the `example/` directory for a complete working example demonstrating logging configuration and level testing.

## Dependencies

- [spf13/cobra](https://github.com/spf13/cobra) - CLI framework
- [spf13/viper](https://github.com/spf13/viper) - Configuration management
- [spf13/pflag](https://github.com/spf13/pflag) - Command-line flag parsing
- [rs/zerolog](https://github.com/rs/zerolog) - Structured logging
- [dioad/util](https://github.com/dioad/util) - Utility functions

## License

See LICENSE file for details.
