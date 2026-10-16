package configcheck_test

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/util"

	"github.com/dioad/cli/configcheck"
)

type common struct {
	LogLevel string `mapstructure:"log-level"`
}

type tlsServer struct {
	CertFile string `mapstructure:"cert-file"`
}

type tunnel struct {
	Name      string     `mapstructure:"name"`
	TLSServer *tlsServer `mapstructure:"tls-server"`
}

type route struct {
	MaxSize int `mapstructure:"max-size"`
}

type agent struct {
	common `mapstructure:",squash"`

	Port    int               `mapstructure:"port"`
	Timeout time.Duration     `mapstructure:"timeout"`
	Secret  util.MaskedString `mapstructure:"secret"`
	BindIP  net.IP            `mapstructure:"bind-ip"`
	Allow   *net.IPNet        `mapstructure:"allow"`
	Tunnels []tunnel          `mapstructure:"tunnels"`
	Routes  map[string]route  `mapstructure:"routes"`
	Extra   map[string]any    `mapstructure:"extra"`
}

func decodeFindings(t *testing.T, input map[string]any) []configcheck.Finding {
	t.Helper()
	var cfg agent
	err := configcheck.Decode(input, &cfg)
	require.Error(t, err)
	var ce *configcheck.Error
	require.ErrorAs(t, err, &ce)
	return ce.Findings
}

func TestDecode_AcceptsKnownKeysThroughTheLoaderHooks(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"log-level": "debug",
		"port":      float64(8080), // as decoded from JSON
		"timeout":   "30s",
		"secret":    "hunter2",
		"bind-ip":   "192.0.2.10",
		"allow":     "10.0.0.0/8",
		"tunnels":   []any{map[string]any{"name": "web", "tls-server": map[string]any{"cert-file": "c.pem"}}},
		"routes":    map[string]any{"dmarc": map[string]any{"max-size": 1024}},
		"extra":     map[string]any{"anything": map[string]any{"goes": true}},
	}

	var cfg agent
	require.NoError(t, configcheck.Decode(input, &cfg))

	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, "hunter2", cfg.Secret.UnmaskedString())
	assert.Equal(t, "192.0.2.10", cfg.BindIP.String())
	assert.Equal(t, "10.0.0.0/8", cfg.Allow.String())
	require.Len(t, cfg.Tunnels, 1)
	assert.Equal(t, "c.pem", cfg.Tunnels[0].TLSServer.CertFile)
	assert.Equal(t, 1024, cfg.Routes["dmarc"].MaxSize)
}

func TestDecode_ReportsUnknownKeysWithTheirFullPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input map[string]any
		want  []configcheck.Finding
	}{
		{
			name:  "root key",
			input: map[string]any{"status-address": ":8080"},
			want:  []configcheck.Finding{{Path: "status-address", Message: "unknown key"}},
		},
		{
			name: "key in a list element",
			input: map[string]any{"tunnels": []any{
				map[string]any{"name": "a"},
				map[string]any{"name": "b", "tls-server-config": map[string]any{}},
			}},
			want: []configcheck.Finding{{Path: "tunnels[1].tls-server-config", Message: "unknown key"}},
		},
		{
			name: "key in a pointer to a struct",
			input: map[string]any{"tunnels": []any{
				map[string]any{"tls-server": map[string]any{"auto-cert-config": map[string]any{}}},
			}},
			want: []configcheck.Finding{{Path: "tunnels[0].tls-server.auto-cert-config", Message: "unknown key"}},
		},
		{
			name:  "key in a map value",
			input: map[string]any{"routes": map[string]any{"dmarc": map[string]any{"max-bytes": 1}}},
			want:  []configcheck.Finding{{Path: "routes[dmarc].max-bytes", Message: "unknown key"}},
		},
		{
			name:  "several keys in one struct, sorted",
			input: map[string]any{"zeta": 1, "alpha": 2},
			want: []configcheck.Finding{
				{Path: "alpha", Message: "unknown key"},
				{Path: "zeta", Message: "unknown key"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, decodeFindings(t, tt.input))
		})
	}
}

func TestDecode_ReportsEveryProblemInOnePass(t *testing.T) {
	t.Parallel()

	findings := decodeFindings(t, map[string]any{
		"port":    "eighty",
		"timeout": "soon",
		"bogus":   true,
		"tunnels": []any{map[string]any{"nmae": "typo"}},
	})

	paths := make([]string, len(findings))
	for i, f := range findings {
		paths[i] = f.Path
	}
	assert.Equal(t, []string{"bogus", "port", "timeout", "tunnels[0].nmae"}, paths)
	assert.Equal(t, "expected type 'int', got unconvertible type 'string'", findings[1].Message)
	assert.Equal(t, "time: invalid duration", findings[2].Message)
}

func TestDecode_SetsFieldsThatDecodeWhenOthersFail(t *testing.T) {
	t.Parallel()

	var cfg agent
	err := configcheck.Decode(map[string]any{"port": 443, "bogus": 1}, &cfg)

	require.Error(t, err)
	assert.Equal(t, 443, cfg.Port)
}

func TestDecode_RejectsANonPointerTargetAsAProgrammingError(t *testing.T) {
	t.Parallel()

	err := configcheck.Decode(map[string]any{}, agent{})

	require.Error(t, err)
	var ce *configcheck.Error
	assert.NotErrorAs(t, err, &ce)
}
