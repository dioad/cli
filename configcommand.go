package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/dioad/config/configcheck"
)

// ConfigCommandOpt is a functional option for NewConfigCommand.
type ConfigCommandOpt func(*configCommand)

type configCommand struct {
	defaultFile string
	example     func() ([]byte, error)
	schema      func() ([]byte, error)
}

// WithConfigCommandFile sets the config file "config validate" checks when it
// is not given --config, normally the file the service loads.
func WithConfigCommandFile(path string) ConfigCommandOpt {
	return func(c *configCommand) { c.defaultFile = path }
}

// WithConfigExample makes "config example" print what example returns, in place
// of the example NewConfigCommand generates from the config struct. Use it when
// a service generates a richer one at build time, with comments taken from the
// struct's doc comments.
func WithConfigExample(example func() ([]byte, error)) ConfigCommandOpt {
	return func(c *configCommand) { c.example = example }
}

// WithConfigSchema makes "config schema" print what schema returns, in place of
// the schema NewConfigCommand generates from the config struct.
func WithConfigSchema(schema func() ([]byte, error)) ConfigCommandOpt {
	return func(c *configCommand) { c.schema = schema }
}

// NewConfigCommand returns a "config" command with three subcommands, for a
// service whose config type is built by newConfig and described as kind (for
// example "accounting"):
//
//   - validate checks a config file without starting the service. It reports
//     every key that matches no setting, with its full path, and every setting
//     the config's Validate method rejects, and exits non-zero if there is any.
//   - example prints an example config file listing every setting.
//   - schema prints the JSON Schema of the config file.
//
// newConfig should return the config with its defaults set, so the example shows
// them. Add the command to the service's root with AddCommand; it reads no
// configuration of its own.
func NewConfigCommand(kind string, newConfig func() Validator, opts ...ConfigCommandOpt) *cobra.Command {
	c := &configCommand{}
	for _, opt := range opts {
		opt(c)
	}
	if c.example == nil {
		c.example = func() ([]byte, error) { return ConfigExample(newConfig()) }
	}
	if c.schema == nil {
		c.schema = func() ([]byte, error) { return ConfigSchema(newConfig()) }
	}

	cmd := &cobra.Command{
		Use:   "config",
		Short: fmt.Sprintf("Check, show and describe the %s config file", kind),
	}
	cmd.AddCommand(c.validateCommand(kind, newConfig), c.exampleCommand(kind), c.schemaCommand(kind))

	return cmd
}

func (c *configCommand) validateCommand(kind string, newConfig func() Validator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: fmt.Sprintf("Check a %s config file for unknown keys and invalid settings", kind),
		Long: fmt.Sprintf(`Check a %[1]s config file without starting the %[1]s.

Reports every key that matches no setting, with its full path (for example a
renamed or misspelled key), and every setting the %[1]s would reject at
startup, all together. Exits non-zero if any problem is found.`, kind),
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ValidateConfigFile(cmd.OutOrStdout(), cmd.Flag("config").Value.String(), newConfig())
		},
	}
	cmd.Flags().StringP("config", "c", c.defaultFile, "config file to check")

	return cmd
}

func (c *configCommand) exampleCommand(kind string) *cobra.Command {
	return &cobra.Command{
		Use:   "example",
		Short: fmt.Sprintf("Print an example %s config file listing every setting", kind),
		Long: fmt.Sprintf(`Print an example %[1]s config file that lists every setting it can have,
each with its description as a comment and its default value, or an empty
value where it has none.

The example is a reference to copy settings from, not a config to run as it
is: some settings must be set to work.`, kind),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return writeOut(cmd.OutOrStdout(), c.example) },
	}
}

func (c *configCommand) schemaCommand(kind string) *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: fmt.Sprintf("Print the JSON Schema of a %s config file", kind),
		Long: fmt.Sprintf(`Print the JSON Schema of a %s config file, for an editor or a validator to
check a file against. The schema checks each key and the type of each value; it
does not apply the rules "config validate" applies beyond that.`, kind),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return writeOut(cmd.OutOrStdout(), c.schema) },
	}
}

func writeOut(w io.Writer, generate func() ([]byte, error)) error {
	out, err := generate()
	if err != nil {
		return err
	}
	_, err = w.Write(out)

	return err
}

// ValidateConfigFile checks the config file at path as the service would at
// startup, without starting it, and writes the result to w: every problem, one
// per line, or a line saying the file is valid. The file is decoded strictly
// into cfg, so a key that matches no field is a problem, and cfg's Validate
// method is then called, so a setting it rejects is too; all are reported
// together. It returns an error if the file is invalid or cannot be read.
func ValidateConfigFile(w io.Writer, path string, cfg Validator) error {
	err := DecodeConfigFile(path, cfg)
	var decodeFindings *configcheck.Error
	if err != nil && !errors.As(err, &decodeFindings) {
		return err
	}

	findings := configcheck.NewCollector()
	findings.Add(err)
	findings.Add(cfg.Validate())

	all := findings.Findings()
	if len(all) == 0 {
		_, err = fmt.Fprintf(w, "%s is valid\n", path)

		return err
	}

	if _, err = fmt.Fprintf(w, "%s:\n", path); err != nil {
		return err
	}
	for _, f := range all {
		if _, err = fmt.Fprintf(w, "  %s\n", f); err != nil {
			return err
		}
	}

	return fmt.Errorf("%s: %d configuration problem(s) found", path, len(all))
}
