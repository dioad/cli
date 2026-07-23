// Package cli_test contains tests for InitConfig, InitViperConfigWithFlagSet,
// and related configuration loading behaviours.
package cli_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli"
)

// writeYAML writes content to a temp file inside t.TempDir() and returns the
// full path. Cleanup is handled automatically by TempDir.
func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600),
		"writeYAML: failed to write temp config file")
	return path
}

// newTestCmd returns a cobra.Command with a --config flag. If configFile is
// non-empty the flag is pre-set to that value, simulating --config=<file>.
func newTestCmd(t *testing.T, configFile string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "testapp"}
	cmd.Flags().StringP("config", "c", "", "config file")
	if configFile != "" {
		require.NoError(t, cmd.Flags().Set("config", configFile),
			"newTestCmd: failed to set --config flag")
	}
	return cmd
}

// TestInitConfig covers the primary behaviours of InitConfig through a
// table-driven set of isolated sub-tests.
//
// Each sub-test uses:
//   - t.TempDir() for filesystem isolation
//   - t.Setenv() for environment variable isolation
//   - WithoutLogging() to avoid global zerolog state changes
//   - WithoutWatchConfig() for backward-compatible option parity (it is now a
//     no-op; InitConfig no longer starts a background watcher goroutine)
func TestInitConfig(t *testing.T) {
	type testConfig struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	}

	tests := []struct {
		name    string
		orgName string
		appName string
		setup   func(t *testing.T) *cobra.Command
		assert  func(t *testing.T, cfg *testConfig, err error)
	}{
		{
			name:    "valid config file is loaded",
			orgName: "testorg",
			appName: "testapp",
			setup: func(t *testing.T) *cobra.Command {
				t.Helper()
				return newTestCmd(t, writeYAML(t, "name: fromfile\nport: 9090\n"))
			},
			assert: func(t *testing.T, cfg *testConfig, err error) {
				require.NoError(t, err, "reading a valid config file should not error")
				assert.Equal(t, "fromfile", cfg.Name,
					"name field should be populated from the YAML file")
				assert.Equal(t, 9090, cfg.Port,
					"port field should be populated from the YAML file")
			},
		},
		{
			name:    "missing config file is not an error",
			orgName: "testorg",
			appName: "testapp",
			setup: func(t *testing.T) *cobra.Command {
				t.Helper()
				// No --config flag set: InitConfig searches standard paths and
				// finds nothing. ConfigFileNotFoundError must be non-fatal.
				return newTestCmd(t, "")
			},
			assert: func(t *testing.T, cfg *testConfig, err error) {
				require.NoError(t, err,
					"absent config file should not cause an error")
			},
		},
		{
			name:    "malformed yaml returns an error",
			orgName: "testorg",
			appName: "testapp",
			setup: func(t *testing.T) *cobra.Command {
				t.Helper()
				return newTestCmd(t, writeYAML(t, "name: [unclosed bracket\n"))
			},
			assert: func(t *testing.T, cfg *testConfig, err error) {
				require.Error(t, err,
					"malformed YAML should return an error")
			},
		},
		{
			name:    "env var overrides config file value",
			orgName: "testorg",
			appName: "testapp",
			setup: func(t *testing.T) *cobra.Command {
				t.Helper()
				// t.Setenv restores the original env var after the test.
				t.Setenv("TESTAPP_NAME", "fromenv")
				return newTestCmd(t, writeYAML(t, "name: fromfile\n"))
			},
			assert: func(t *testing.T, cfg *testConfig, err error) {
				require.NoError(t, err,
					"env var override should not cause an error")
				assert.Equal(t, "fromenv", cfg.Name,
					"env var TESTAPP_NAME should override the config file value")
			},
		},
		{
			name:    "invalid org name returns error",
			orgName: "org name with spaces",
			appName: "testapp",
			setup: func(t *testing.T) *cobra.Command {
				t.Helper()
				return newTestCmd(t, "")
			},
			assert: func(t *testing.T, _ *testConfig, err error) {
				require.Error(t, err,
					"invalid org name should return an error")
				assert.ErrorContains(t, err, "orgName",
					"error message should identify orgName as the problem")
			},
		},
		{
			name:    "invalid app name returns error",
			orgName: "testorg",
			appName: "app/name",
			setup: func(t *testing.T) *cobra.Command {
				t.Helper()
				return newTestCmd(t, "")
			},
			assert: func(t *testing.T, _ *testConfig, err error) {
				require.Error(t, err,
					"invalid app name should return an error")
				assert.ErrorContains(t, err, "appName",
					"error message should identify appName as the problem")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cmd := tt.setup(t)
			var cfg testConfig

			// Act
			_, err := cli.InitConfig(tt.orgName, tt.appName, cmd, &cfg,
				cli.WithoutLogging(),     // keep global zerolog state unchanged
				cli.WithoutWatchConfig(), // no-op, kept for option parity
			)

			// Assert
			tt.assert(t, &cfg, err)
		})
	}
}

// TestInitConfigDoesNotLeakWatcherGoroutine verifies that InitConfig no
// longer starts a background fsnotify watcher goroutine per call (the
// watcher was removed because it had no safe way to propagate reloaded
// config into the caller's struct — see WithoutWatchConfig's doc comment).
func TestInitConfigDoesNotLeakWatcherGoroutine(t *testing.T) {
	type testConfig struct {
		Name string `mapstructure:"name"`
	}

	before := runtime.NumGoroutine()

	for range 10 {
		cmd := newTestCmd(t, writeYAML(t, "name: x\n"))
		var cfg testConfig
		_, err := cli.InitConfig("testorg", "testapp", cmd, &cfg, cli.WithoutLogging())
		require.NoError(t, err)
	}

	runtime.Gosched()
	after := runtime.NumGoroutine()
	assert.LessOrEqual(t, after, before+2,
		"InitConfig should not start a background watcher goroutine per call")
}

// TestInitViperConfigWithFlagSet verifies that a custom pflag.FlagSet is
// correctly bound and unmarshalled into the target struct.
//
// Note: InitViperConfigWithFlagSet uses the global Viper instance. These
// tests must not run in parallel to avoid interference.
func TestInitViperConfigWithFlagSet(t *testing.T) {
	type serverConfig struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	}

	t.Run("flag values are unmarshalled into config struct", func(t *testing.T) {
		// Arrange
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flags.String("host", "localhost", "server host")
		flags.Int("port", 8080, "server port")
		require.NoError(t, flags.Parse([]string{"--host=myserver", "--port=3000"}),
			"flag parse should not fail")

		var cfg serverConfig

		// Act
		err := cli.InitViperConfigWithFlagSet("testorg", "testapp", &cfg, flags)

		// Assert
		require.NoError(t, err, "valid flags should unmarshal without error")
		assert.Equal(t, "myserver", cfg.Host,
			"host should be populated from the --host flag")
		assert.Equal(t, 3000, cfg.Port,
			"port should be populated from the --port flag")
	})

	t.Run("env var overrides flag default", func(t *testing.T) {
		// Arrange
		t.Setenv("TESTAPP2_HOST", "fromenv")

		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flags.String("host", "localhost", "server host")
		require.NoError(t, flags.Parse(nil), "flag parse should not fail")

		var cfg serverConfig

		// Act
		err := cli.InitViperConfigWithFlagSet("testorg", "testapp2", &cfg, flags)

		// Assert
		require.NoError(t, err, "env var override should not error")
		assert.Equal(t, "fromenv", cfg.Host,
			"TESTAPP2_HOST should override the --host flag default")
	})
}
