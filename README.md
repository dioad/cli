# dioad/cli

A Go library that provides utilities for building command-line applications with configuration management, logging, and Cobra integration.

## Features

- **Cobra Integration**: Simplified command creation with type-safe configuration
- **Configuration Management**: Flexible configuration loading from multiple sources (files, environment variables, flags)
- **Logging**: Built-in logging setup with zerolog, file rotation support via lumberjack
- **Docker Support**: Automatic detection and configuration path handling for Docker environments
- **Type Safety**: Generic functions for type-safe command handlers

## Installation

```bash
go get github.com/dioad/cli
```

## Quick Start

Here's a simple example of creating a CLI application:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/dioad/cli"
    "github.com/spf13/cobra"
)

type Config struct {
    Name string `mapstructure:"name"`
    Port int    `mapstructure:"port"`
}

func main() {
    rootCmd := &cobra.Command{
        Use:   "myapp",
        Short: "My application",
    }
    
    // Set up context with org and app names
    ctx := cli.Context(
        context.Background(),
        cli.SetOrgName("myorg"),
        cli.SetAppName("myapp"),
    )
    rootCmd.SetContext(ctx)
    
    // Create a command with configuration
    defaultConfig := &Config{
        Name: "default",
        Port: 8080,
    }
    
    serveCmd := cli.NewCommand(
        &cobra.Command{
            Use:   "serve",
            Short: "Start the server",
        },
        func(ctx context.Context, cfg *Config) error {
            fmt.Printf("Starting server %s on port %d\n", cfg.Name, cfg.Port)
            return nil
        },
        defaultConfig,
        cli.WithConfigFlag("~/.config/myorg/myapp/config.yaml"),
    )
    
    rootCmd.AddCommand(serveCmd)
    rootCmd.Execute()
}
```

## Configuration

The library supports loading configuration from multiple sources in the following priority order (highest to lowest):

1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

### Configuration File Locations

The library searches for configuration files in these locations:

- `/etc/{orgName}/{appName}/`
- `$HOME/.{orgName}/{appName}/`
- `$HOME/.config/{orgName}/{appName}/`
- Current directory (`.`)
- Custom path specified via `--config` flag

### Docker Support

When running in Docker (detected by presence of `/.dockerenv`), the library uses:
- `/config` for configuration files
- `/persist` for persistent data

### Environment Variables

Environment variables are automatically mapped from configuration keys:
- Prefix: Application name (uppercase)
- Separators: Hyphens and dots are converted to underscores
- Example: `MYAPP_LOG_LEVEL` maps to `log.level`

## Logging Configuration

The library includes built-in logging configuration using zerolog:

```go
type Config struct {
    cli.CommonConfig
}
```

Logging options in your config file:

```yaml
log:
  level: debug           # trace, debug, info, warn, error, fatal
  file: /var/log/app.log # Optional: log to file instead of stdout
  max-size: 100          # Max size in MB before rotation
  max-age: 28            # Max age in days to keep old logs
  max-backups: 3         # Max number of old log files to retain
  use-local-time: true   # Use local time instead of UTC
  compress: true         # Compress rotated logs
```

## API Overview

### Command Creation

**`NewCommand[T any]`**: Create a new Cobra command with type-safe configuration handling.

```go
cmd := cli.NewCommand(
    cobraCmd *cobra.Command,
    runFunc func(context.Context, *T) error,
    defaultConfig *T,
    opts ...CommandOpt,
)
```

### Configuration Functions

- **`InitConfig`**: Initialize configuration for a command
- **`InitViperConfig`**: Initialize Viper configuration with org and app names
- **`DefaultConfigPath`**: Get the default configuration path
- **`DefaultConfigFile`**: Get the default configuration file path
- **`DefaultPersistencePath`**: Get the default persistence path

### Context Management

- **`Context`**: Create a context with org/app names
- **`SetOrgName`**: Set organization name in context
- **`SetAppName`**: Set application name in context

### Helper Functions

- **`CobraRunEWithConfig`**: Create a Cobra RunE function with config initialization
- **`IsDocker`**: Check if running in Docker environment

## Examples

### Command with Config File

```go
cmd := cli.NewCommand(
    &cobra.Command{
        Use:   "serve",
        Short: "Start server",
    },
    runServer,
    &ServerConfig{},
    cli.WithConfigFlag("~/.config/myorg/myapp/server.yaml"),
)
```

### Multiple Commands

```go
rootCmd := &cobra.Command{Use: "myapp"}
ctx := cli.Context(
    context.Background(),
    cli.SetOrgName("myorg"),
    cli.SetAppName("myapp"),
)
rootCmd.SetContext(ctx)

rootCmd.AddCommand(
    cli.NewCommand(&cobra.Command{Use: "serve"}, runServe, &ServeConfig{}),
    cli.NewCommand(&cobra.Command{Use: "migrate"}, runMigrate, &MigrateConfig{}),
)
```

## Dependencies

This library uses:
- [spf13/cobra](https://github.com/spf13/cobra) - Command-line interface
- [spf13/viper](https://github.com/spf13/viper) - Configuration management
- [rs/zerolog](https://github.com/rs/zerolog) - Structured logging
- [urfave/sflags](https://github.com/urfave/sflags) - Flag parsing
- [natefinch/lumberjack](https://gopkg.in/natefinch/lumberjack.v2) - Log rotation

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
