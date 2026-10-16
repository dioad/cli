package configcheck_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli/configcheck"
)

func TestCollector_DerivedCollectorsReportUnderTheirPathAndShareFindings(t *testing.T) {
	t.Parallel()

	root := configcheck.NewCollector()
	tunnels := root.Field("traffic").Field("tunnels")
	tunnels.Index(2).Field("subdomain").Errorf("must be a DNS label")
	root.Field("mail").Key("dmarc").Errorf("unknown handler type %q", "dmark")
	root.Errorf("no tunnels configured")

	assert.Equal(t, []configcheck.Finding{
		{Path: "traffic.tunnels[2].subdomain", Message: "must be a DNS label"},
		{Path: "mail[dmarc]", Message: `unknown handler type "dmark"`},
		{Path: "", Message: "no tunnels configured"},
	}, tunnels.Findings())
}

func TestCollector_AddPrefixesTheFindingsOfAnError(t *testing.T) {
	t.Parallel()

	inner := &configcheck.Error{Findings: []configcheck.Finding{
		{Path: "port", Message: "unknown key"},
		{Path: "[0].name", Message: "is required"},
		{Path: "", Message: "is empty"},
	}}

	c := configcheck.NewCollector()
	c.Field("tunnels").Index(1).Field("config").Add(fmt.Errorf("decoding: %w", inner))

	assert.Equal(t, []configcheck.Finding{
		{Path: "tunnels[1].config.port", Message: "unknown key"},
		{Path: "tunnels[1].config[0].name", Message: "is required"},
		{Path: "tunnels[1].config", Message: "is empty"},
	}, c.Findings())
}

func TestCollector_AddTurnsAPlainErrorIntoOneFinding(t *testing.T) {
	t.Parallel()

	c := configcheck.NewCollector()
	c.Field("listen-address").Add(errors.New("missing port in address"))
	c.Add(nil)

	assert.Equal(t, []configcheck.Finding{
		{Path: "listen-address", Message: "missing port in address"},
	}, c.Findings())
}

func TestCollector_ErrIsNilWithoutFindings(t *testing.T) {
	t.Parallel()

	assert.NoError(t, configcheck.NewCollector().Err())
}

func TestCollector_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var c configcheck.Collector
	c.Field("name").Errorf("is required")

	err := c.Err()
	require.Error(t, err)
	assert.Equal(t, "name: is required", err.Error())
}

func TestCollector_IsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()

	c := configcheck.NewCollector()
	done := make(chan struct{})
	for i := range 10 {
		go func() {
			defer func() { done <- struct{}{} }()
			c.Field("tunnels").Index(i).Errorf("bad")
		}()
	}
	for range 10 {
		<-done
	}

	assert.Len(t, c.Findings(), 10)
}

func TestError_ListsOneFindingPerLine(t *testing.T) {
	t.Parallel()

	err := &configcheck.Error{Findings: []configcheck.Finding{
		{Path: "a.b", Message: "unknown key"},
		{Message: "no tunnels configured"},
	}}

	assert.Equal(t, "a.b: unknown key\nno tunnels configured", err.Error())
}

func TestJoinPath(t *testing.T) {
	t.Parallel()

	tests := []struct{ base, rel, want string }{
		{"", "name", "name"},
		{"tunnels[0]", "", "tunnels[0]"},
		{"tunnels", "[0].name", "tunnels[0].name"},
		{"traffic", "tunnels", "traffic.tunnels"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, configcheck.JoinPath(tt.base, tt.rel), "JoinPath(%q, %q)", tt.base, tt.rel)
	}
}

func ExampleCollector() {
	type tunnel struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	}
	raw := []map[string]any{
		{"name": "web", "port": 8080},
		{"port": 22, "tls-server-config": map[string]any{}},
	}

	c := configcheck.NewCollector()
	tunnels := c.Field("tunnels")
	for i, entry := range raw {
		var t tunnel
		entryCollector := tunnels.Index(i)
		entryCollector.Add(configcheck.Decode(entry, &t))
		if t.Name == "" {
			entryCollector.Field("name").Errorf("is required")
		}
	}

	fmt.Println(c.Err())
	// Output:
	// tunnels[1].tls-server-config: unknown key
	// tunnels[1].name: is required
}
