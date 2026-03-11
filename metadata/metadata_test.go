package metadata_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli"
	"github.com/dioad/cli/metadata"
)

// captureStdout redirects os.Stdout to a pipe, calls f, then returns
// everything written to the pipe as a string.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	origStdout := os.Stdout
	os.Stdout = w

	f()

	os.Stdout = origStdout
	w.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	r.Close()

	return buf.String()
}

func newTestContext() context.Context {
	return cli.Context(
		context.Background(),
		cli.SetOrgName("testorg"),
		cli.SetAppName("testapp"),
	)
}

// TestNewVersionCommand_FlagRegistered verifies that the --json flag is
// registered on the command returned by NewVersionCommand.
func TestNewVersionCommand_FlagRegistered(t *testing.T) {
	info := metadata.BuildInfo{
		Version: "1.2.3",
		Commit:  "abc1234",
		Date:    "2024-01-01",
	}

	cmd := metadata.NewVersionCommand("myorg", "myapp", info)

	jsonFlag := cmd.Flags().Lookup("json")
	require.NotNil(t, jsonFlag, "--json flag must be registered on the version command")
	assert.Equal(t, "false", jsonFlag.DefValue, "--json flag default should be false")
}

// TestNewVersionCommand_PlainOutput verifies that the command prints a single
// "orgName appName version" line when --json is not set.
func TestNewVersionCommand_PlainOutput(t *testing.T) {
	info := metadata.BuildInfo{
		Version: "1.2.3",
		Commit:  "abc1234",
		Date:    "2024-01-01",
	}

	cmd := metadata.NewVersionCommand("myorg", "myapp", info)
	cmd.SetArgs([]string{})
	cmd.SilenceUsage = true

	var execErr error
	got := captureStdout(t, func() {
		execErr = cmd.ExecuteContext(newTestContext())
	})

	require.NoError(t, execErr)
	assert.Equal(t, "myorg myapp 1.2.3\n", got)
}

// TestNewVersionCommand_JSONOutput verifies that the command prints valid JSON
// containing version, commit, and date fields when --json is passed.
func TestNewVersionCommand_JSONOutput(t *testing.T) {
	info := metadata.BuildInfo{
		Version: "1.2.3",
		Commit:  "abc1234",
		Date:    "2024-01-01",
	}

	cmd := metadata.NewVersionCommand("myorg", "myapp", info)
	cmd.SetArgs([]string{"--json"})
	cmd.SilenceUsage = true

	var execErr error
	got := captureStdout(t, func() {
		execErr = cmd.ExecuteContext(newTestContext())
	})

	require.NoError(t, execErr)

	var result map[string]string
	require.NoError(t, json.Unmarshal([]byte(got), &result), "output should be valid JSON")
	assert.Equal(t, info.Version, result["version"])
	assert.Equal(t, info.Commit, result["commit"])
	assert.Equal(t, info.Date, result["date"])
}
