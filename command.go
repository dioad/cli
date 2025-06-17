package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/urfave/sflags"
	"github.com/urfave/sflags/gen/gpflag"
)

func NewCommand[T any](cmd *cobra.Command, runFunc func(context.Context, *T) error, defaultConfig *T, opts ...CommandOpt) *cobra.Command {
	cmd.RunE = CobraRunEWithConfig(runFunc, defaultConfig)

	_ = gpflag.ParseTo(defaultConfig, cmd.Flags(), sflags.InheritDeprecated(), sflags.InheritHidden())

	for _, opt := range opts {
		opt(cmd)
	}

	return cmd
}

type CommandOpt func(*cobra.Command)

func WithConfigFlag(defaultConfigFile string) func(*cobra.Command) {
	return func(cmd *cobra.Command) {
		cmd.Flags().StringP("config", "c", defaultConfigFile, "config file")
	}
}
