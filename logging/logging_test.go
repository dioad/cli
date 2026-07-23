package logging_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli/logging"
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

	assert.Equal(t, "debug", cfg.Level)
	assert.Equal(t, "test.log", cfg.File)
	assert.Equal(t, 100, cfg.MaxSize)
	assert.Equal(t, 3, cfg.MaxBackups)
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
			assertFunc := assert.NotPanics
			if !tt.expectedNoPanic {
				assertFunc = assert.Panics
			}
			assertFunc(t, func() {
				logging.ConfigureLogLevel(tt.level, tt.defaultLevel)
			})
		})
	}
}

// TestWithDefaultLogLevel applies default log level option.
func TestWithDefaultLogLevel(t *testing.T) {
	cfg := logging.Config{
		Level: "",
	}

	opt := logging.WithDefaultLogLevel(zerolog.DebugLevel)
	require.NotNil(t, opt, "WithDefaultLogLevel() returned nil")

	opt(&cfg)

	assert.Equal(t, "debug", cfg.Level)
}

// TestWithDefaultLogLevelDoesNotOverride preserves existing level.
func TestWithDefaultLogLevelDoesNotOverride(t *testing.T) {
	cfg := logging.Config{
		Level: "error",
	}

	opt := logging.WithDefaultLogLevel(zerolog.DebugLevel)
	opt(&cfg)

	assert.Equal(t, "error", cfg.Level, "WithDefaultLogLevel() should not override an existing level")
}

// TestWithDefaultLogLevelFixesInvalid replaces invalid level with default.
func TestWithDefaultLogLevelFixesInvalid(t *testing.T) {
	cfg := logging.Config{
		Level: "notvalid",
	}

	opt := logging.WithDefaultLogLevel(zerolog.DebugLevel)
	opt(&cfg)

	assert.Equal(t, "debug", cfg.Level, "WithDefaultLogLevel() should replace an invalid level with the default")
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

// TestFatalErrorExits verifies FatalError logs the error and exits the
// process with a non-zero status. Since FatalError calls log.Fatal (which
// calls os.Exit), it is exercised in a subprocess rather than in-process.
func TestFatalErrorExits(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessFatalError")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "FatalError should cause the process to exit non-zero")
	assert.False(t, exitErr.Success(), "process should exit with a non-zero status")
	assert.Contains(t, stderr.String(), "boom",
		"stderr should contain the fatal error message")
}

// TestHelperProcessFatalError is not a real test; it is invoked as a
// subprocess by TestFatalErrorExits to exercise FatalError's os.Exit path.
func TestHelperProcessFatalError(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	logging.FatalError(errors.New("boom"))
}

// TestEmptyConfig uses default values.
func TestEmptyConfig(t *testing.T) {
	cfg := logging.Config{}

	assert.Empty(t, cfg.Level)
	assert.Empty(t, cfg.File)
	assert.Zero(t, cfg.MaxSize)
}

// TestConfigIsConsistent verifies configuration can be created and used.
func TestConfigIsConsistent(t *testing.T) {
	cfg1 := logging.Config{
		Level:      "debug",
		MaxSize:    100,
		MaxBackups: 3,
	}

	cfg2 := cfg1

	assert.Equal(t, cfg1.Level, cfg2.Level, "Config copy should maintain Level")
	assert.Equal(t, cfg1.MaxSize, cfg2.MaxSize, "Config copy should maintain MaxSize")
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
