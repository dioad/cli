package cli_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli"
	"github.com/dioad/cli/logging"
)

// TestCommandTreeExecution demonstrates a complete integration scenario:
// a root command with a subcommand is built and actually executed, verifying
// that the execFunc is called (Inspiring, Behavioral).
func TestCommandTreeExecution(t *testing.T) {
	type AppConfig struct {
		cli.CommonConfig
		AppName string `mapstructure:"app-name"`
		Version string `mapstructure:"version"`
	}

	cfg := &AppConfig{
		AppName: "testapp",
		Version: "1.0.0",
		CommonConfig: cli.CommonConfig{
			Logging: logging.Config{Level: "info"},
		},
	}

	var called bool

	rootCmd := &cobra.Command{
		Use:          "testapp",
		Short:        "Test application",
		SilenceUsage: true,
	}

	ctx := cli.Context(
		context.Background(),
		cli.SetOrgName("testorg"),
		cli.SetAppName("testapp"),
	)

	subCmd := cli.NewCommand(
		&cobra.Command{Use: "action", Short: "Perform an action"},
		func(ctx context.Context, c *AppConfig) error {
			called = true
			assert.NotEmpty(t, c.AppName, "AppName should be populated from flag defaults")
			return nil
		},
		cfg,
	)
	rootCmd.AddCommand(subCmd)

	// Act — actually execute the command tree, not just inspect its structure.
	rootCmd.SetArgs([]string{"action"})
	err := rootCmd.ExecuteContext(ctx)

	// Assert
	require.NoError(t, err, "executing the action subcommand should not error")
	assert.True(t, called, "execFunc should have been called during command execution")
}

// TestCommandWithDefaultConfig verifies that execFunc receives the config
// values derived from flag defaults when no explicit flags are provided.
func TestCommandWithDefaultConfig(t *testing.T) {
	type ServerConfig struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	}

	defaults := &ServerConfig{
		Host: "localhost",
		Port: 8080,
	}

	var gotHost string
	var gotPort int

	cmd := cli.NewCommand(
		&cobra.Command{
			Use:          "serve",
			SilenceUsage: true,
		},
		func(ctx context.Context, c *ServerConfig) error {
			gotHost = c.Host
			gotPort = c.Port
			return nil
		},
		defaults,
	)

	ctx := cli.Context(
		context.Background(),
		cli.SetOrgName("testorg"),
		cli.SetAppName("testapp"),
	)

	// Act — execute the command with no explicit flag values.
	cmd.SetArgs([]string{})
	err := cmd.ExecuteContext(ctx)

	// Assert
	require.NoError(t, err, "executing the command with defaults should not error")
	assert.Equal(t, defaults.Host, gotHost,
		"host should match the default value registered from the config struct")
	assert.Equal(t, defaults.Port, gotPort,
		"port should match the default value registered from the config struct")
}

// TestCobraRunERequiresContext verifies that CobraRunE returns an error rather
// than calling os.Exit when org/app names are absent from the context.
func TestCobraRunERequiresContext(t *testing.T) {
	type Config struct{}
	var called bool

	cmd := cli.NewCommand(
		&cobra.Command{
			Use:           "test",
			SilenceUsage:  true,
			SilenceErrors: true,
		},
		func(ctx context.Context, c *Config) error {
			called = true
			return nil
		},
		&Config{},
	)

	// No org/app name — plain background context.
	cmd.SetArgs([]string{})
	err := cmd.ExecuteContext(context.Background())

	assert.Error(t, err,
		"CobraRunE should return an error when org/app name is not set in context")
	assert.ErrorContains(t, err, "org name",
		"error should mention that org name is missing")
	assert.False(t, called,
		"execFunc should not have been called when context is incomplete")
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

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(ctx)

	require.NotNil(t, cmd.Context(), "context should be set on the command")
	assert.Equal(t, orgName, cli.OrgNameFromContext(cmd.Context()),
		"org name should be retrievable from the command context")
	assert.Equal(t, appName, cli.AppNameFromContext(cmd.Context()),
		"app name should be retrievable from the command context")
}

// TestCobraRunEInjectsLoggerIntoContext verifies that CobraRunE attaches the
// logger configured by InitConfig to the context passed to execFunc, so
// execFunc can retrieve it via zerolog.Ctx(ctx) instead of the global logger.
func TestCobraRunEInjectsLoggerIntoContext(t *testing.T) {
	type Config struct{}
	var level zerolog.Level

	cmd := cli.NewCommand(
		&cobra.Command{Use: "test", SilenceUsage: true},
		func(ctx context.Context, c *Config) error {
			level = zerolog.Ctx(ctx).GetLevel()
			return nil
		},
		&Config{},
	)

	ctx := cli.Context(
		context.Background(),
		cli.SetOrgName("testorg"),
		cli.SetAppName("testapp"),
	)

	cmd.SetArgs([]string{})
	require.NoError(t, cmd.ExecuteContext(ctx))
	assert.NotEqual(t, zerolog.Disabled, level,
		"zerolog.Ctx(ctx) inside execFunc should return the configured logger, not the no-op logger")
}

// TestEnvironmentIntegration verifies that environment variables are
// correctly read and applied by InitViperConfigWithFlagSet.
func TestEnvironmentIntegration(t *testing.T) {
	type Config struct {
		Debug bool   `mapstructure:"debug"`
		Host  string `mapstructure:"host"`
	}

	// Arrange — use t.Setenv so the env vars are automatically cleaned up.
	t.Setenv("ENVTEST_DEBUG", "true")
	t.Setenv("ENVTEST_HOST", "env-host")

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Bool("debug", false, "enable debug")
	flags.String("host", "localhost", "host")
	require.NoError(t, flags.Parse(nil), "flag parse should not fail")

	var cfg Config

	// Act
	err := cli.InitViperConfigWithFlagSet("testorg", "envtest", &cfg, flags)

	// Assert
	require.NoError(t, err, "env var configuration should not error")
	assert.True(t, cfg.Debug,
		"ENVTEST_DEBUG=true should set debug to true")
	assert.Equal(t, "env-host", cfg.Host,
		"ENVTEST_HOST should override the default flag value")
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
			func(ctx context.Context, c *Config) error { return nil },
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
