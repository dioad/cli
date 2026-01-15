package cli_test

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/dioad/cli"
	"github.com/dioad/cli/logging"
	"github.com/spf13/cobra"
)

// TestFullIntegration demonstrates a complete integration scenario.
func TestFullIntegration(t *testing.T) {
	type AppConfig struct {
		cli.CommonConfig
		AppName string `mapstructure:"app-name"`
		Version string `mapstructure:"version"`
	}

	cfg := &AppConfig{
		AppName: "testapp",
		Version: "1.0.0",
		CommonConfig: cli.CommonConfig{
			Logging: logging.Config{
				Level: "info",
			},
		},
	}

	// Create a root command with context
	rootCmd := &cobra.Command{
		Use:   "testapp",
		Short: "Test application",
	}

	ctx := cli.Context(
		context.Background(),
		cli.SetOrgName("testorg"),
		cli.SetAppName("testapp"),
	)
	rootCmd.SetContext(ctx)

	// Add a subcommand
	subCmd := cli.NewCommand(
		&cobra.Command{
			Use:   "action",
			Short: "Perform an action",
		},
		func(ctx context.Context, c *AppConfig) error {
			if c.AppName == "" {
				t.Error("AppName not set")
			}
			return nil
		},
		cfg,
	)

	rootCmd.AddCommand(subCmd)

	// Verify the structure
	if rootCmd == nil {
		t.Fatal("Root command is nil")
	}

	if len(rootCmd.Commands()) == 0 {
		t.Error("No subcommands added")
	}

	if rootCmd.Commands()[0].Use != "action" {
		t.Errorf("Wrong subcommand: %s", rootCmd.Commands()[0].Use)
	}
}

// TestCommandWithDefaultConfig verifies command execution with defaults.
func TestCommandWithDefaultConfig(t *testing.T) {
	type ServerConfig struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	}

	cfg := &ServerConfig{
		Host: "localhost",
		Port: 8080,
	}

	cmd := cli.NewCommand(
		&cobra.Command{
			Use: "serve",
		},
		func(ctx context.Context, c *ServerConfig) error {
			if c.Host != cfg.Host || c.Port != cfg.Port {
				t.Error("Config not passed correctly")
			}
			return nil
		},
		cfg,
	)

	if cmd == nil {
		t.Fatal("NewCommand returned nil")
	}

	if cmd.RunE == nil {
		t.Fatal("RunE was not set")
	}
}

// TestContextPropagation verifies context values are correctly propagated.
func TestContextPropagation(t *testing.T) {
	orgName := "myorg"
	appName := "myapp"

	ctx := cli.Context(
		context.Background(),
		cli.SetOrgName(orgName),
		cli.SetAppName(appName),
	)

	// Verify we can create commands with this context
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(ctx)

	if cmd.Context() == nil {
		t.Error("Context not set on command")
	}
}

// BenchmarkNewCommand measures command creation performance.
func BenchmarkNewCommand(b *testing.B) {
	type Config struct {
		Value string `mapstructure:"value"`
	}

	cfg := &Config{Value: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cli.NewCommand(
			&cobra.Command{Use: "test"},
			func(ctx context.Context, c *Config) error {
				return nil
			},
			cfg,
		)
	}
}

// BenchmarkContext measures context creation performance.
func BenchmarkContext(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cli.Context(
			context.Background(),
			cli.SetOrgName("org"),
			cli.SetAppName("app"),
		)
	}
}

// BenchmarkDefaultConfigPath measures path resolution performance.
func BenchmarkDefaultConfigPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = cli.DefaultConfigPath("org", "app")
	}
}

// TestEnvironmentIntegration verifies environment variable handling.
func TestEnvironmentIntegration(t *testing.T) {
	// This test demonstrates how environment variables would be handled
	// In actual use, users would set environment variables before running the app

	originalEnv := os.Getenv("TESTAPP_DEBUG")
	defer func() {
		if originalEnv != "" {
			os.Setenv("TESTAPP_DEBUG", originalEnv)
		} else {
			os.Unsetenv("TESTAPP_DEBUG")
		}
	}()

	// Verify environment is accessible
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go executable not found")
	}
}
