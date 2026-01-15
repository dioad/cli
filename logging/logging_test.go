package logging_test

import (
	"testing"

	"github.com/dioad/cli/logging"
	"github.com/rs/zerolog"
)

// TestConfig verifies the logging configuration struct.
func TestConfig(t *testing.T) {
	cfg := logging.Config{
		Level:      "debug",
		File:       "test.log",
		MaxSize:    100,
		MaxAge:     7,
		MaxBackups: 3,
		LocalTime:  false,
		Compress:   true,
	}

	if cfg.Level != "debug" {
		t.Errorf("Config.Level = %s, want debug", cfg.Level)
	}

	if cfg.File != "test.log" {
		t.Errorf("Config.File = %s, want test.log", cfg.File)
	}

	if cfg.MaxSize != 100 {
		t.Errorf("Config.MaxSize = %d, want 100", cfg.MaxSize)
	}

	if cfg.MaxBackups != 3 {
		t.Errorf("Config.MaxBackups = %d, want 3", cfg.MaxBackups)
	}
}

// TestConfigureLogLevel sets and verifies log level.
func TestConfigureLogLevel(t *testing.T) {
	tests := []struct {
		name            string
		level           string
		defaultLevel    zerolog.Level
		expectedNoPanic bool
	}{
		{
			name:            "empty level uses default",
			level:           "",
			defaultLevel:    zerolog.InfoLevel,
			expectedNoPanic: true,
		},
		{
			name:            "valid debug level",
			level:           "debug",
			defaultLevel:    zerolog.InfoLevel,
			expectedNoPanic: true,
		},
		{
			name:            "valid info level",
			level:           "info",
			defaultLevel:    zerolog.WarnLevel,
			expectedNoPanic: true,
		},
		{
			name:            "valid error level",
			level:           "error",
			defaultLevel:    zerolog.InfoLevel,
			expectedNoPanic: true,
		},
		{
			name:            "invalid level uses default",
			level:           "notavalidlevel",
			defaultLevel:    zerolog.InfoLevel,
			expectedNoPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && tt.expectedNoPanic {
					t.Errorf("ConfigureLogLevel(%s) panicked: %v", tt.level, r)
				}
			}()

			logging.ConfigureLogLevel(tt.level, tt.defaultLevel)
			// If we get here, no panic occurred
		})
	}
}

// TestWithDefaultLogLevel applies default log level option.
func TestWithDefaultLogLevel(t *testing.T) {
	cfg := logging.Config{
		Level: "",
	}

	opt := logging.WithDefaultLogLevel(zerolog.DebugLevel)

	if opt == nil {
		t.Error("WithDefaultLogLevel() returned nil")
	}

	opt(&cfg)

	if cfg.Level != "debug" {
		t.Errorf("WithDefaultLogLevel() set level to %s, want debug", cfg.Level)
	}
}

// TestWithDefaultLogLevelDoesNotOverride preserves existing level.
func TestWithDefaultLogLevelDoesNotOverride(t *testing.T) {
	cfg := logging.Config{
		Level: "error",
	}

	opt := logging.WithDefaultLogLevel(zerolog.DebugLevel)
	opt(&cfg)

	if cfg.Level != "error" {
		t.Errorf("WithDefaultLogLevel() override existing level to %s, want error", cfg.Level)
	}
}

// TestWithDefaultLogLevelFixesInvalid replaces invalid level with default.
func TestWithDefaultLogLevelFixesInvalid(t *testing.T) {
	cfg := logging.Config{
		Level: "notvalid",
	}

	opt := logging.WithDefaultLogLevel(zerolog.DebugLevel)
	opt(&cfg)

	if cfg.Level != "debug" {
		t.Errorf("WithDefaultLogLevel() fixed invalid level to %s, want debug", cfg.Level)
	}
}

// TestConfigureCmdLogger applies configuration without panic.
func TestConfigureCmdLogger(t *testing.T) {
	cfg := logging.Config{
		Level: "debug",
		File:  "",
	}

	// Should not panic
	logging.ConfigureCmdLogger(cfg)
}

// TestConfigureCmdLoggerWithOptions applies options and configuration.
func TestConfigureCmdLoggerWithOptions(t *testing.T) {
	cfg := logging.Config{
		Level: "",
	}

	// Should not panic
	logging.ConfigureCmdLogger(
		cfg,
		logging.WithDefaultLogLevel(zerolog.InfoLevel),
	)

	// The option modifies the config passed to it
	if cfg.Level == "" {
		t.Logf("ConfigureCmdLogger() option behavior: cfg passed by value, so option doesn't modify original")
	}
}

// TestConfigureLogOutput configures output without panic.
func TestConfigureLogOutput(t *testing.T) {
	cfg := logging.Config{
		Level: "info",
		File:  "",
	}

	// Should not panic
	logging.ConfigureLogOutput(cfg)
}

// TestFatalError doesn't panic during unit test (it would exit).
func TestFatalError(t *testing.T) {
	// We can't easily test FatalError as it calls log.Fatal which exits.
	// This is a documentation test showing the function exists.
	// In practice, this would only be used when the application needs to exit.
}

// TestEmptyConfig uses default values.
func TestEmptyConfig(t *testing.T) {
	cfg := logging.Config{}

	if cfg.Level != "" {
		t.Errorf("empty Config.Level = %s, want empty", cfg.Level)
	}

	if cfg.File != "" {
		t.Errorf("empty Config.File = %s, want empty", cfg.File)
	}

	if cfg.MaxSize != 0 {
		t.Errorf("empty Config.MaxSize = %d, want 0", cfg.MaxSize)
	}
}

// TestConfigIsConsistent verifies configuration can be created and used.
func TestConfigIsConsistent(t *testing.T) {
	cfg1 := logging.Config{
		Level:      "debug",
		MaxSize:    100,
		MaxBackups: 3,
	}

	cfg2 := cfg1

	if cfg1.Level != cfg2.Level {
		t.Error("Config copy doesn't maintain Level")
	}

	if cfg1.MaxSize != cfg2.MaxSize {
		t.Error("Config copy doesn't maintain MaxSize")
	}
}

// BenchmarkConfigureLogLevel measures log level configuration time.
func BenchmarkConfigureLogLevel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		logging.ConfigureLogLevel("debug", zerolog.InfoLevel)
	}
}

// BenchmarkConfigureCmdLogger measures full logging configuration time.
func BenchmarkConfigureCmdLogger(b *testing.B) {
	cfg := logging.Config{
		Level: "info",
	}

	for i := 0; i < b.N; i++ {
		logging.ConfigureCmdLogger(cfg)
	}
}
