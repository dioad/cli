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
	assert.Equal(t, path+" is valid\n", out.String())
}
