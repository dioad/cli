package cli

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/urfave/sflags"
	"github.com/urfave/sflags/gen/gpflag"
)

// NewCommand creates a new Cobra command with type-safe configuration handling.
//
// It wraps the provided Cobra command with automatic configuration loading,
// populating flags from the config struct, and executing the command with
// configuration management.
func NewCommand[T any](cmd *cobra.Command, runFunc func(context.Context, *T) error, defaultConfig *T, opts ...CommandOpt) *cobra.Command {
	cmd.RunE = CobraRunE(runFunc)

	if err := gpflag.ParseTo(defaultConfig, cmd.Flags(), sflags.InheritDeprecated(), sflags.InheritHidden()); err != nil {
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
