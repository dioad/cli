package metadata_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli/metadata"
)

func TestNewVersionCommand_PlainText(t *testing.T) {
	info := metadata.BuildInfo{
		Version: "1.2.3",
		Commit:  "abc1234",
		Date:    "2024-01-01",
	}

	cmd := metadata.NewVersionCommand("myorg", "myapp", info)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "myorg myapp 1.2.3\n", buf.String())
}

func TestNewVersionCommand_JSON(t *testing.T) {
	info := metadata.BuildInfo{
		Version: "1.2.3",
		Commit:  "abc1234",
		Date:    "2024-01-01",
	}

	cmd := metadata.NewVersionCommand("myorg", "myapp", info)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	var got map[string]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, info.Version, got["version"])
	assert.Equal(t, info.Commit, got["commit"])
	assert.Equal(t, info.Date, got["date"])
}

func TestNewVersionCommand_NoInitConfig(t *testing.T) {
	// This test verifies the command executes without requiring org/app context,
	// confirming it bypasses CobraRunE / InitConfig entirely.
	info := metadata.BuildInfo{Version: "0.1.0", Commit: "XX", Date: "1970-01-01"}

	cmd := metadata.NewVersionCommand("org", "app", info)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err, "version command should run without context or config")
	assert.True(t, strings.HasPrefix(buf.String(), "org app "),
		"output should start with org and app name")
}
