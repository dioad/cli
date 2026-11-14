package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli"
)

type tlsBlock struct {
	CertFile string `desc:"path to the certificate" mapstructure:"cert-file"`
}

type serviceConfig struct {
	ListenAddress string            `desc:"address to listen on" mapstructure:"listen-address"`
	Workers       int               `desc:"number of workers" mapstructure:"workers"`
	DrainDelay    time.Duration     `desc:"how long to keep serving after a stop request" mapstructure:"drain-delay"`
	Origins       []string          `desc:"origins allowed by CORS" mapstructure:"cors-allowed-origins"`
	TLS           tlsBlock          `mapstructure:"tls-server"`
	Labels        map[string]string `desc:"free-form labels" mapstructure:"labels"`
	Ignored       string            `mapstructure:"-"`
}

func newServiceConfig() cli.Validator {
	return &serviceConfig{ListenAddress: ":8080", Workers: 4, DrainDelay: 5 * time.Second}
}

func (c *serviceConfig) Validate() error {
	if c.Workers < 1 {
		return errors.New("workers must be at least 1")
	}

	return nil
}

func runConfigCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "svc"}
	root.AddCommand(cli.NewConfigCommand("svc", newServiceConfig))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"config"}, args...))
	err := root.Execute()

	return out.String(), err
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/svc.yaml"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestConfigValidate_AcceptsAGoodFile(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "listen-address: ':9000'\nworkers: 2\n")

	out, err := runConfigCommand(t, "validate", "--config", path)

	require.NoError(t, err)
	assert.Equal(t, path+" is valid\n", out)
}

func TestConfigValidate_ReportsEveryProblemTogether(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "workers: 0\nlisten-adress: ':1'\ntls-server:\n  cert: x\n")

	out, err := runConfigCommand(t, "validate", "--config", path)

	require.ErrorContains(t, err, "3 configuration problem(s) found")
	assert.Contains(t, out, "listen-adress", "a misspelt key is named")
	assert.Contains(t, out, "tls-server.cert", "an unknown nested key is reported with its full path")
	assert.Contains(t, out, "workers must be at least 1", "a setting Validate rejects is reported with them")
}

func TestConfigValidate_ReportsAFileThatCannotBeRead(t *testing.T) {
	t.Parallel()

	_, err := runConfigCommand(t, "validate", "--config", t.TempDir()+"/missing.yaml")

	require.ErrorContains(t, err, "reading config file")
}

func TestConfigValidate_UsesTheDefaultFile(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "workers: 3\n")
	root := &cobra.Command{Use: "svc"}
	root.AddCommand(cli.NewConfigCommand("svc", newServiceConfig, cli.WithConfigCommandFile(path)))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"config", "validate"})

	require.NoError(t, root.Execute())
	assert.Equal(t, path+" is valid\n", out.String())
}

func TestConfigExample_ListsEverySettingWithItsDescriptionAndDefault(t *testing.T) {
	t.Parallel()

	out, err := runConfigCommand(t, "example")

	require.NoError(t, err)
	assert.Equal(t, `# address to listen on
listen-address: :8080
# number of workers
workers: 4
# how long to keep serving after a stop request
drain-delay: 5s
# origins allowed by CORS
cors-allowed-origins: []
tls-server:
    # path to the certificate
    cert-file: ""
# free-form labels
labels: {}
`, out)
}

func TestConfigExample_IsAValidConfigFile(t *testing.T) {
	t.Parallel()
	out, err := runConfigCommand(t, "example")
	require.NoError(t, err)

	validated, err := runConfigCommand(t, "validate", "--config", writeConfig(t, out))

	require.NoError(t, err, validated)
}

func TestConfigSchema_DescribesEachKeyAndRejectsOthers(t *testing.T) {
	t.Parallel()

	out, err := runConfigCommand(t, "schema")

	require.NoError(t, err)
	var schema struct {
		Type                 string                    `json:"type"`
		AdditionalProperties bool                      `json:"additionalProperties"`
		Properties           map[string]map[string]any `json:"properties"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &schema))
	assert.Equal(t, "object", schema.Type)
	assert.False(t, schema.AdditionalProperties, "an unknown key is rejected")
	assert.Equal(t, "string", schema.Properties["listen-address"]["type"])
	assert.Equal(t, "address to listen on", schema.Properties["listen-address"]["description"])
	assert.Equal(t, "integer", schema.Properties["workers"]["type"])
	assert.Equal(t, "string", schema.Properties["drain-delay"]["type"], "a duration is written as a string")
	assert.Equal(t, "array", schema.Properties["cors-allowed-origins"]["type"])
	assert.Equal(t, "object", schema.Properties["tls-server"]["type"])
	assert.NotContains(t, schema.Properties, "Ignored")
}

func TestConfigCommand_ExampleAndSchemaCanBeSuppliedByTheService(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "svc"}
	root.AddCommand(cli.NewConfigCommand("svc", newServiceConfig,
		cli.WithConfigExample(func() ([]byte, error) { return []byte("custom example\n"), nil }),
		cli.WithConfigSchema(func() ([]byte, error) { return []byte("{}\n"), nil }),
	))

	for args, want := range map[string]string{"example": "custom example\n", "schema": "{}\n"} {
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"config", args})
		require.NoError(t, root.Execute())
		assert.Equal(t, want, out.String())
	}
}

func ExampleNewConfigCommand() {
	root := &cobra.Command{Use: "svc"}
	root.AddCommand(cli.NewConfigCommand("svc", newServiceConfig))

	file, _ := os.CreateTemp("", "svc-*.yaml")
	defer func() { _ = os.Remove(file.Name()) }()
	_, _ = file.WriteString("workers: 0\nlisten-adress: ':1'\n")
	_ = file.Close()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"config", "validate", "--config", file.Name()})
	_ = root.Execute()

	fmt.Print(strings.ReplaceAll(out.String(), file.Name(), "svc.yaml"))
	// Output:
	// svc.yaml:
	//   listen-adress: unknown key
	//   workers must be at least 1
	// Error: svc.yaml: 2 configuration problem(s) found
}
