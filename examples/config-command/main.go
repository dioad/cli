// Command config-command shows a service command with the "config validate",
// "config example" and "config schema" subcommands that cli.NewConfigCommand
// provides. Its run command prints its configuration and exits, so the example
// has no long-lived process.
//
// See README.md for the output on a bad config file.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/dioad/cli"
)

// Config is the greeter's configuration. Every field has a desc tag, which is
// its flag's help text, its comment in "config example" and its description in
// "config schema".
type Config struct {
	Greeting   string        `desc:"what to say" mapstructure:"greeting"`
	Repeat     int           `desc:"how many times to say it" mapstructure:"repeat"`
	DrainDelay time.Duration `desc:"how long to keep running after a stop request" mapstructure:"drain-delay"`
	Management Management    `mapstructure:"management"`
}

// Management is a nested block: its keys are management.listen-address in the
// file and --management-listen-address on the command line.
type Management struct {
	ListenAddress string `desc:"address of the management listener" mapstructure:"listen-address"`
}

// NewConfig returns the configuration with its defaults set, which is what
// "config example" shows.
func NewConfig() *Config {
	return &Config{Greeting: "hello", Repeat: 1, DrainDelay: 5 * time.Second, Management: Management{ListenAddress: "127.0.0.1:4444"}}
}

// Validate rejects settings the service cannot run with. "config validate"
// reports these together with any unknown keys.
func (c *Config) Validate() error {
	if c.Repeat < 1 {
		return fmt.Errorf("repeat must be at least 1, got %d", c.Repeat)
	}

	return nil
}

func run(_ context.Context, cfg *Config) error {
	for range cfg.Repeat {
		fmt.Println(cfg.Greeting)
	}

	return nil
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{Use: "greeter", Short: "Say hello"}

	serve := cli.NewCommand(&cobra.Command{Use: "run", Short: "Say the greeting"}, run, NewConfig(),
		cli.WithConfigFlag(""), cli.WithConfigValidation())
	root.AddCommand(serve)
	root.AddCommand(cli.NewConfigCommand("greeter", func() cli.Validator { return NewConfig() }))

	return root
}

func main() {
	ctx := cli.Context(context.Background(), cli.SetOrgName("dioad"), cli.SetAppName("greeter"))
	if err := newRootCommand().ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
