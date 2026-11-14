package cli_test

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli"
)

type rateLimitBlock struct {
	RateLimitRPS float64 `desc:"requests per second" mapstructure:"rate-limit-rps"`
}

type managementBlock struct {
	ListenAddress string `desc:"management listen address" mapstructure:"listen-address"`
}

// flagKeysConfig has the three shapes issue #18 reports: a named struct
// squashed into its parent, a field whose key differs from its Go name, and a
// nested block.
type flagKeysConfig struct {
	RateLimit      rateLimitBlock  `desc:"" mapstructure:",squash"`
	AllowedOrigins []string        `desc:"origins allowed by CORS" mapstructure:"cors-allowed-origins"`
	Management     managementBlock `mapstructure:"management"`
	Plain          string          `desc:"a plain field" mapstructure:"plain"`
	Chosen         string          `desc:"named by its flag tag" flag:"picked" mapstructure:"chosen-key"`
}

func newFlagKeysCmd() *cobra.Command {
	return cli.NewCommand(&cobra.Command{Use: "testapp"}, func(context.Context, *flagKeysConfig) error { return nil }, &flagKeysConfig{})
}

func TestNewCommand_FlagsAreNamedAfterConfigKeys(t *testing.T) {
	t.Parallel()
	cmd := newFlagKeysCmd()

	for _, name := range []string{"rate-limit-rps", "cors-allowed-origins", "management-listen-address", "plain", "picked"} {
		assert.NotNil(t, cmd.Flags().Lookup(name), "--%s should exist", name)
	}
	assert.Nil(t, cmd.Flags().Lookup("chosen-key"), "a flag tag name wins over the mapstructure key")
}

func TestNewCommand_OldFlagNamesStayAsDeprecatedAliases(t *testing.T) {
	t.Parallel()
	cmd := newFlagKeysCmd()

	for oldName, newName := range map[string]string{
		"rate-limit-rate-limit-rps": "rate-limit-rps",
		"allowed-origins":           "cors-allowed-origins",
		"management-listen-address": "management-listen-address",
	} {
		if oldName == newName {
			continue
		}
		alias := cmd.Flags().Lookup(oldName)
		require.NotNil(t, alias, "--%s should still be accepted", oldName)
		assert.True(t, alias.Hidden, "--%s is hidden", oldName)
		assert.Contains(t, alias.Deprecated, "--"+newName)
	}
}

// TestInitConfig_EveryGeneratedFlagReachesTheConfig is the check issue #18
// asks for: set each generated flag and confirm the decoded struct changed, so
// a flag named for a key nothing decodes cannot go unnoticed.
func TestInitConfig_EveryGeneratedFlagReachesTheConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		flag  string
		value string
		check func(t *testing.T, cfg *flagKeysConfig)
	}{
		{"rate-limit-rps", "5", func(t *testing.T, c *flagKeysConfig) { assert.InDelta(t, 5.0, c.RateLimit.RateLimitRPS, 0) }},
		{"cors-allowed-origins", "https://a.example", func(t *testing.T, c *flagKeysConfig) {
			assert.Equal(t, []string{"https://a.example"}, c.AllowedOrigins)
		}},
		{"management-listen-address", ":4444", func(t *testing.T, c *flagKeysConfig) { assert.Equal(t, ":4444", c.Management.ListenAddress) }},
		{"plain", "p", func(t *testing.T, c *flagKeysConfig) { assert.Equal(t, "p", c.Plain) }},
		{"picked", "c", func(t *testing.T, c *flagKeysConfig) { assert.Equal(t, "c", c.Chosen) }},
		// The previous names keep working.
		{"rate-limit-rate-limit-rps", "7", func(t *testing.T, c *flagKeysConfig) { assert.InDelta(t, 7.0, c.RateLimit.RateLimitRPS, 0) }},
		{"allowed-origins", "https://b.example", func(t *testing.T, c *flagKeysConfig) {
			assert.Equal(t, []string{"https://b.example"}, c.AllowedOrigins)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			t.Parallel()
			cmd := newFlagKeysCmd()
			require.NoError(t, cmd.Flags().Parse([]string{"--" + tt.flag + "=" + tt.value}))

			var cfg flagKeysConfig
			_, err := cli.InitConfig("testorg", "testapp", cmd, &cfg, cli.WithoutLogging())

			require.NoError(t, err)
			tt.check(t, &cfg)
		})
	}
}

func TestInitConfig_FlagOverridesConfigFileAtTheDecodedKey(t *testing.T) {
	t.Parallel()
	cmd := newFlagKeysCmd()
	cmd.Flags().StringP("config", "c", "", "config file")
	require.NoError(t, cmd.Flags().Parse([]string{
		"--config=" + writeYAML(t, "management:\n  listen-address: ':1'\nrate-limit-rps: 1\n"),
		"--rate-limit-rps=9",
	}))

	var cfg flagKeysConfig
	_, err := cli.InitConfig("testorg", "testapp", cmd, &cfg, cli.WithoutLogging())

	require.NoError(t, err)
	assert.InDelta(t, 9.0, cfg.RateLimit.RateLimitRPS, 0, "the flag beats the file")
	assert.Equal(t, ":1", cfg.Management.ListenAddress, "an unset flag leaves the file's value")
}

func TestInitConfig_EnvironmentOverridesUseTheConfigKey(t *testing.T) {
	t.Setenv("TESTAPP_MANAGEMENT_LISTEN_ADDRESS", ":5555")
	cmd := newFlagKeysCmd()

	var cfg flagKeysConfig
	_, err := cli.InitConfig("testorg", "testapp", cmd, &cfg, cli.WithoutLogging())

	require.NoError(t, err)
	assert.Equal(t, ":5555", cfg.Management.ListenAddress)
}
