package logadapter_test

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli/logadapter"
)

func newAdapter(t *testing.T, level zerolog.Level) (logadapter.Adapter, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf)
	return logadapter.New(logger, level), buf
}

func TestAdapter_Write(t *testing.T) {
	t.Parallel()

	a, buf := newAdapter(t, zerolog.DebugLevel)

	n, err := a.Write([]byte("dial failed\n"))

	require.NoError(t, err)
	assert.Equal(t, len("dial failed\n"), n)
	assert.Contains(t, buf.String(), `"message":"dial failed"`)
	assert.Contains(t, buf.String(), `"level":"debug"`)
}

func TestAdapter_Printf(t *testing.T) {
	t.Parallel()

	a, buf := newAdapter(t, zerolog.ErrorLevel)

	a.Printf("no route matched %q", "example.com")

	assert.Contains(t, buf.String(), `"message":"no route matched \"example.com\""`)
	assert.Contains(t, buf.String(), `"level":"error"`)
}

func TestAdapter_Println(t *testing.T) {
	t.Parallel()

	a, buf := newAdapter(t, zerolog.ErrorLevel)

	a.Println("dial error:", "connection refused")

	assert.Contains(t, buf.String(), `"message":"dial error: connection refused"`)
}

func TestAdapter_Errorf(t *testing.T) {
	t.Parallel()

	a, buf := newAdapter(t, zerolog.ErrorLevel)

	a.Errorf("socks command %s failed", "connect")

	assert.Contains(t, buf.String(), `"message":"socks command connect failed"`)
}
