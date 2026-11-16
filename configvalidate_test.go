package cli_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli"
	"github.com/dioad/config/configcheck"
)

// secretConfig needs a secret that a deployment supplies through the
// environment, leaving the key empty in the file.
type secretConfig struct {
	Keycloak struct {
		ClientSecret string `mapstructure:"client-secret"`
	} `mapstructure:"keycloak"`
}

func (c *secretConfig) Validate() error {
	if c.Keycloak.ClientSecret == "" {
		return errors.New("keycloak.client-secret is required")
	}

	return nil
}

func newSecretConfig() cli.Validator { return &secretConfig{} }

const (
	secretApp     = "connect-control"
	secretEnvVar  = "CONNECT_CONTROL_KEYCLOAK_CLIENT_SECRET"
	secretMissing = "keycloak:\n  client-secret: \"\"\n"
)

func withSecretEnv() []cli.ConfigCommandOpt {
	return []cli.ConfigCommandOpt{cli.WithConfigEnvPrefix(secretApp)}
}

func TestConfigValidate_AppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv(secretEnvVar, "from-env")
	path := writeConfig(t, secretMissing)

	out, err := runConfigCommandFor(t, newSecretConfig, withSecretEnv(), "validate", "--config", path)

	require.NoError(t, err, out)
	assert.Equal(t, path+" is valid (environment overrides applied)\n", out)
}

func TestConfigValidate_ReportsASettingTheEnvironmentDoesNotSupply(t *testing.T) {
	t.Setenv(secretEnvVar, "")
	path := writeConfig(t, secretMissing)

	out, err := runConfigCommandFor(t, newSecretConfig, withSecretEnv(), "validate", "--config", path)

	require.ErrorContains(t, err, "1 configuration problem(s) found (environment overrides applied)")
	assert.Contains(t, out, "keycloak.client-secret is required")
}

func TestConfigValidate_IgnoreEnvChecksTheFileAlone(t *testing.T) {
	t.Setenv(secretEnvVar, "from-env")
	path := writeConfig(t, secretMissing)

	out, err := runConfigCommandFor(t, newSecretConfig, withSecretEnv(), "validate", "--config", path, "--ignore-env")

	require.ErrorContains(t, err, "1 configuration problem(s) found (environment overrides not applied)")
	assert.Contains(t, out, "keycloak.client-secret is required")
}

func TestConfigValidate_WithoutAnEnvPrefixChecksTheFileAlone(t *testing.T) {
	t.Setenv(secretEnvVar, "from-env")
	path := writeConfig(t, secretMissing)

	out, err := runConfigCommandFor(t, newSecretConfig, nil, "validate", "--config", path)

	require.ErrorContains(t, err, "(environment overrides not applied)")
	assert.Contains(t, out, "keycloak.client-secret is required")
}

func TestConfigValidate_AVariableMatchingNoSettingIsNotAProblem(t *testing.T) {
	t.Setenv("CONNECT_CONTROL_NO_SUCH_SETTING", "stray")
	t.Setenv("CONNECT_CONTROL_KEYCLOAK_CLIENT_SECRT", "misspelt")
	path := writeConfig(t, "keycloak:\n  client-secret: in-file\n")

	out, err := runConfigCommandFor(t, newSecretConfig, withSecretEnv(), "validate", "--config", path)

	require.NoError(t, err, out)
}

// The service loads its config with InitConfig; "config validate" must read
// the same variable for the same key, including the "-" and "." in the app
// name and the key.
func TestConfigValidate_ReadsTheVariableInitConfigReads(t *testing.T) {
	t.Setenv(secretEnvVar, "from-env")
	path := writeConfig(t, secretMissing)

	var loaded secretConfig
	_, err := cli.InitConfig("dioad", secretApp, newTestCmd(t, path), &loaded, cli.WithoutLogging())
	require.NoError(t, err)
	require.Equal(t, "from-env", loaded.Keycloak.ClientSecret, "InitConfig takes the secret from the variable")

	out, err := runConfigCommandFor(t, newSecretConfig, withSecretEnv(), "validate", "--config", path)

	require.NoError(t, err, out)
}

func TestConfigValidate_HelpSaysWhetherTheEnvironmentIsApplied(t *testing.T) {
	t.Parallel()

	t.Run("with an env prefix", func(t *testing.T) {
		t.Parallel()
		out, err := runConfigCommandFor(t, newSecretConfig, withSecretEnv(), "validate", "--help")

		require.NoError(t, err)
		assert.Contains(t, out, "Environment variables override the file's settings")
		assert.Contains(t, out, "CONNECT_CONTROL_SECTION_SOME_KEY overrides\nsection.some-key")
		assert.Contains(t, out, "--ignore-env")
	})

	t.Run("without one", func(t *testing.T) {
		t.Parallel()
		out, err := runConfigCommandFor(t, newSecretConfig, nil, "validate", "--help")

		require.NoError(t, err)
		assert.Contains(t, out, "Checks the file alone")
		assert.NotContains(t, out, "--ignore-env")
	})
}

// referringConfig refers to files whose problems only checking them finds.
type referringConfig struct {
	Workers        int      `mapstructure:"workers"`
	DiscoveryFiles []string `mapstructure:"discovery-files"`

	referenceErr error
}

func (c *referringConfig) Validate() error {
	if c.Workers < 1 {
		return errors.New("workers must be at least 1")
	}

	return nil
}

func (c *referringConfig) CheckReferences(findings *configcheck.Collector) error {
	if len(c.DiscoveryFiles) == 0 {
		findings.Field("discovery-files").Errorf("no file provides a tunnel type")
	}

	return c.referenceErr
}

func TestValidateConfigFile_ReportsProblemsUnderTheFileTheyAreIn(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "workers: 0\n")
	cfg := &referringConfig{referenceErr: errors.Join(
		&cli.FileError{Path: "discovery/web.yaml", Err: &configcheck.Error{Findings: []configcheck.Finding{
			{Path: "tunnels[0].port", Message: "unknown key"},
			{Path: "tunnels[1]", Message: "has no type"},
		}}},
		&cli.FileError{Path: "discovery/db.yaml", Err: errors.New("is not YAML")},
	)}
	var out bytes.Buffer

	err := cli.ValidateConfigFile(&out, path, cfg)

	require.ErrorContains(t, err, path+": 5 configuration problem(s) found")
	assert.Equal(t, path+`:
  workers must be at least 1
  discovery-files: no file provides a tunnel type
discovery/web.yaml:
  tunnels[0].port: unknown key
  tunnels[1]: has no type
discovery/db.yaml:
  is not YAML
`, out.String())
}

func TestValidateConfigFile_ReportsOnlyAReferencedFilesProblems(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "workers: 1\ndiscovery-files: [discovery/web.yaml]\n")
	cfg := &referringConfig{referenceErr: &cli.FileError{Path: "discovery/web.yaml", Err: errors.New("is not YAML")}}
	var out bytes.Buffer

	err := cli.ValidateConfigFile(&out, path, cfg)

	require.ErrorContains(t, err, "1 configuration problem(s) found")
	assert.Equal(t, "discovery/web.yaml:\n  is not YAML\n", out.String())
}

func TestValidateConfigFile_AReferenceThatCannotBeCheckedIsAProblemWithTheConfig(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "workers: 1\ndiscovery-files: [discovery/web.yaml]\n")
	cfg := &referringConfig{referenceErr: errors.New("reading plugin manifests: permission denied")}
	var out bytes.Buffer

	err := cli.ValidateConfigFile(&out, path, cfg)

	require.ErrorContains(t, err, "1 configuration problem(s) found")
	assert.Equal(t, path+":\n  reading plugin manifests: permission denied\n", out.String())
}

func TestValidateConfigFile_AcceptsAConfigWhoseReferencesAreGood(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "workers: 1\ndiscovery-files: [discovery/web.yaml]\n")
	var out bytes.Buffer

	err := cli.ValidateConfigFile(&out, path, &referringConfig{})

	require.NoError(t, err)
	assert.Equal(t, path+" is valid (environment overrides not applied)\n", out.String())
}
