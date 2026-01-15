// Package cli provides utilities for building CLI applications with Cobra and Viper.
//
// It simplifies the creation of command-line applications by integrating Cobra's command
// framework with Viper's configuration management, providing sensible defaults for:
//
// - Configuration loading from files, environment variables, and command-line flags
// - Type-safe command execution using Go generics
// - Structured logging via zerolog
// - Path management with Docker support
// - Context propagation for application metadata
//
// # Basic Usage
//
// Create a configuration struct:
//
//	type ServerConfig struct {
//		Port int    `mapstructure:"port"`
//		Host string `mapstructure:"host"`
//	}
//
// Create a command handler:
//
//	func serveCommand(ctx context.Context, cfg *ServerConfig) error {
//		fmt.Printf("Starting server on %s:%d\n", cfg.Host, cfg.Port)
//		return nil
//	}
//
// Register the command with NewCommand:
//
//	cfg := &ServerConfig{Port: 8080, Host: "localhost"}
//	cmd := cli.NewCommand(
//		&cobra.Command{
//			Use:   "serve",
//			Short: "Start the server",
//		},
//		serveCommand,
//		cfg,
//		cli.WithConfigFlag("config.yaml"),
//	)
//
// # Configuration File Format
//
// Configuration files are YAML with support for multiple sections:
//
//	port: 8080
//	host: localhost
//	log:
//	  level: debug
//	  file: "~/.app/logs/app.log"
//	  max-size: 100
//	  compress: true
//
// # Configuration Loading Order
//
// Configurations are loaded and merged in this order (last wins):
//
// 1. Config file at /etc/{org}/{app}/config.yaml
// 2. Config file at $HOME/.config/{org}/{app}/config.yaml
// 3. Command-line flags
// 4. Environment variables with prefix {APPNAME}_
// 5. Config file specified by --config flag
//
// # Context Management
//
// Store application metadata in context for propagation through command execution:
//
//	ctx := cli.Context(
//		context.Background(),
//		cli.SetOrgName("myorg"),
//		cli.SetAppName("myapp"),
//	)
//	cmd.SetContext(ctx)
//
// # Advanced Features
//
// Path helpers detect and adapt to Docker environments:
//
//	configPath, err := cli.DefaultConfigPath("myorg", "myapp")
//	// Returns /config in Docker, $HOME/.config/myorg/myapp otherwise
//
// Error handling patterns follow Go best practices with wrapped errors:
//
//	err := cli.InitViperConfig("org", "app", cfg)
//	if err != nil {
//		return fmt.Errorf("initialization failed: %w", err)
//	}
//
// # See Also
//
// Package logging: provides structured logging configuration via zerolog.
package cli
