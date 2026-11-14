package cli

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/urfave/sflags"
)

// NewCommand creates a new Cobra command with type-safe configuration handling.
//
// It wraps the provided Cobra command with automatic configuration loading,
// populating flags from the config struct, and executing the command with
// configuration management.
//
// A flag is named after its field's mapstructure key path, so --log-level sets
// log.level; see registerConfigFlags. Set a flag tag name on a field to choose
// its flag name instead.
func NewCommand[T any](cmd *cobra.Command, runFunc func(context.Context, *T) error, defaultConfig *T, opts ...CommandOpt) *cobra.Command {
	cmd.RunE = CobraRunE(runFunc)

	if err := registerConfigFlags(cmd.Flags(), defaultConfig, sflags.InheritDeprecated(), sflags.InheritHidden()); err != nil {
		panic(fmt.Sprintf("cli: failed to register flags from config type: %v", err))
	}

	for _, opt := range opts {
		opt(cmd)
	}

	return cmd
}

// CommandOpt is a functional option for customizing a Cobra command.
type CommandOpt func(*cobra.Command)

// WithConfigFlag adds a --config/-c flag to the command for specifying a config file path.
func WithConfigFlag(defaultConfigFile string) func(*cobra.Command) {
	return func(cmd *cobra.Command) {
		cmd.Flags().StringP("config", "c", defaultConfigFile, "config file")
	}
}

// WithGlobalZerologLevel returns a CommandOpt that sets the global zerolog log level.
func WithGlobalZerologLevel(level zerolog.Level) CommandOpt {
	return func(_ *cobra.Command) {
		zerolog.SetGlobalLevel(level)
	}
}

// UndocumentedFlags returns the visible flags in cmd's tree that have no usage
// text, as "command path --flag" strings. A flag a user can see in --help
// without a description is a defect, so a consumer can guard its CLI with
//
//	require.Empty(t, cli.UndocumentedFlags(rootCmd))
//
// Hidden and deprecated flags are not reported, since --help does not list them.
func UndocumentedFlags(cmd *cobra.Command) []string {
	var missing []string
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Usage == "" && !f.Hidden && f.Deprecated == "" {
			missing = append(missing, cmd.CommandPath()+" --"+f.Name)
		}
	})
	for _, child := range cmd.Commands() {
		missing = append(missing, UndocumentedFlags(child)...)
	}

	return missing
}
