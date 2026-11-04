package cli

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nestedFlagBlock struct {
	Address string `desc:"listen address" mapstructure:"address"`
	Port    int    `desc:"listen port"    mapstructure:"port"`
}

type inheritFlagsConfig struct {
	Visible    string          `desc:"a visible flag"    mapstructure:"visible"`
	Hidden     nestedFlagBlock `flag:",hidden"           mapstructure:"hidden"`
	Deprecated nestedFlagBlock `flag:",deprecated"       mapstructure:"deprecated"`
}

// TestNewCommand_NestedFlagsInheritParentTags pins the behaviour consumers
// rely on to keep whole config blocks out of --help: a hidden or deprecated
// tag on a struct field applies to every flag generated from its fields.
func TestNewCommand_NestedFlagsInheritParentTags(t *testing.T) {
	t.Parallel()

	cmd := NewCommand(&cobra.Command{Use: "test"}, func(context.Context, *inheritFlagsConfig) error { return nil }, &inheritFlagsConfig{})

	tests := []struct {
		flag           string
		wantHidden     bool
		wantDeprecated bool
	}{
		{flag: "visible"},
		{flag: "hidden-address", wantHidden: true},
		{flag: "hidden-port", wantHidden: true},
		{flag: "deprecated-address", wantDeprecated: true},
		{flag: "deprecated-port", wantDeprecated: true},
	}
	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			t.Parallel()

			f := cmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f, "flag --%s was not registered", tt.flag)
			assert.Equal(t, tt.wantHidden, f.Hidden, "hidden")
			assert.Equal(t, tt.wantDeprecated, f.Deprecated != "", "deprecated")
		})
	}
}
