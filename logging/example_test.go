package logging_test

import (
	"fmt"

	"github.com/dioad/cli/logging"
	"github.com/rs/zerolog"
)

// ExampleConfig demonstrates creating a logging configuration.
func ExampleConfig() {
	cfg := logging.Config{
		Level:      "debug",
		File:       "~/.app/logs/app.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
	}

	fmt.Printf("Logging level: %s\n", cfg.Level)
	fmt.Printf("Max file size: %d MB\n", cfg.MaxSize)
	// Output:
	// Logging level: debug
	// Max file size: 100 MB
}

// ExampleConfigureLogLevel demonstrates setting the log level.
func ExampleConfigureLogLevel() {
	logging.ConfigureLogLevel("info", zerolog.WarnLevel)
	fmt.Println("Log level configured to info")
	// Output: Log level configured to info
}

// ExampleWithDefaultLogLevel demonstrates using a default log level option.
func ExampleWithDefaultLogLevel() {
	cfg := logging.Config{
		Level: "",
	}

	opt := logging.WithDefaultLogLevel(zerolog.DebugLevel)
	opt(&cfg)

	fmt.Printf("Default level applied: %s\n", cfg.Level)
	// Output: Default level applied: debug
}

// ExampleConfigureCmdLogger demonstrates configuring the command logger.
func ExampleConfigureCmdLogger() {
	cfg := logging.Config{
		Level:      "info",
		MaxSize:    100,
		MaxBackups: 3,
	}

	logging.ConfigureCmdLogger(cfg)
	fmt.Println("Command logger configured")
	// Output: Command logger configured
}

// ExampleConfigureLogOutput demonstrates configuring log output.
func ExampleConfigureLogOutput() {
	cfg := logging.Config{
		Level: "debug",
	}

	logging.ConfigureLogOutput(cfg)
	fmt.Println("Log output configured")
	// Output: Log output configured
}
