package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()

	return out.String(), err
}

func TestExample_ConfigValidateReportsABadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "greeter.yaml")
	require.NoError(t, os.WriteFile(path, []byte("greeting: hi\nrepeat: 0\nmanagement:\n  listen-adress: ':1'\n"), 0o600))

	out, err := execute(t, "config", "validate", "--config", path)

	require.Error(t, err)
	assert.Contains(t, out, "management.listen-adress: unknown key")
	assert.Contains(t, out, "repeat must be at least 1, got 0")
}

func TestExample_ConfigExampleShowsDefaults(t *testing.T) {
	out, err := execute(t, "config", "example")

	require.NoError(t, err)
	assert.Contains(t, out, "# what to say\ngreeting: hello\n")
	assert.Contains(t, out, "listen-address: 127.0.0.1:4444")
}

func TestExample_FlagNamesFollowTheConfigKeys(t *testing.T) {
	root := newRootCommand()
	runCmd, _, err := root.Find([]string{"run"})
	require.NoError(t, err)

	assert.NotNil(t, runCmd.Flags().Lookup("management-listen-address"))
	assert.NotNil(t, runCmd.Flags().Lookup("drain-delay"))
}
