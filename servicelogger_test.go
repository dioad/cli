package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli"
)

func logLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var line map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line), "log output: %s", buf.String())

	return line
}

func TestServiceLogger_AddsTheServiceField(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ctx := zerolog.New(&buf).With().Str("request", "r1").Logger().WithContext(context.Background())

	zerolog.Ctx(cli.ServiceLogger(ctx, "accounting")).Error().Msg("hello")

	line := logLine(t, &buf)
	assert.Equal(t, "accounting", line["service"])
	assert.Equal(t, "r1", line["request"], "fields already on the logger are kept")
}

func TestServiceLogger_DoesNotChangeTheCallersContext(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ctx := zerolog.New(&buf).WithContext(context.Background())

	_ = cli.ServiceLogger(ctx, "accounting")
	zerolog.Ctx(ctx).Error().Msg("hello")

	assert.NotContains(t, logLine(t, &buf), "service")
}

func TestServiceLogger_LeavesZerologGlobalsAlone(t *testing.T) {
	before, level, format := zerolog.DefaultContextLogger, zerolog.GlobalLevel(), zerolog.TimeFieldFormat

	_ = cli.ServiceLogger(zerolog.New(&bytes.Buffer{}).WithContext(context.Background()), "accounting")

	assert.Same(t, before, zerolog.DefaultContextLogger)
	assert.Equal(t, level, zerolog.GlobalLevel())
	assert.Equal(t, format, zerolog.TimeFieldFormat)
}

func TestServiceLogger_CanBeCalledConcurrentlyForSeveralServices(t *testing.T) {
	t.Parallel()
	var bufs [4]bytes.Buffer
	done := make(chan struct{})
	for i := range bufs {
		go func() {
			defer func() { done <- struct{}{} }()
			ctx := zerolog.New(&bufs[i]).WithContext(context.Background())
			ctx = cli.ServiceLogger(ctx, fmt.Sprintf("svc-%d", i))
			zerolog.Ctx(ctx).Error().Msg("up")
		}()
	}
	for range bufs {
		<-done
	}

	for i := range bufs {
		assert.Equal(t, fmt.Sprintf("svc-%d", i), logLine(t, &bufs[i])["service"], "each service keeps its own label")
	}
}

func ExampleServiceLogger() {
	var out bytes.Buffer
	ctx := zerolog.New(&out).WithContext(context.Background())

	before := zerolog.DefaultContextLogger
	ctx = cli.ServiceLogger(ctx, "accounting")
	// Error level, so the line is written whatever global level other code
	// in the process has set.
	zerolog.Ctx(ctx).Error().Msg("failed to start")

	fmt.Print(out.String())
	fmt.Println("globals untouched:", zerolog.DefaultContextLogger == before)
	// Output:
	// {"level":"error","service":"accounting","message":"failed to start"}
	// globals untouched: true
}
