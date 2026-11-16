package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// ConfigCommandOpt is a functional option for NewConfigCommand.
type ConfigCommandOpt func(*configCommand)

type configCommand struct {
	defaultFile    string
	envPrefix      string
	example        func() ([]byte, error)
	schema         func() ([]byte, error)
	schemaVariants []schemaVariant
	validateHelp   string
	exampleHelp    string
	schemaHelp     string
}

// schemaVariant is a schema "config schema" prints in place of the config
// file's when the flag named after it is given.
type schemaVariant struct {
	name   string
	usage  string
	schema func() ([]byte, error)
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

// WithConfigSchemaVariant adds a --name flag, described by usage, to "config
// schema" that makes it print what schema returns in place of the config
// file's schema. Use it for a second kind of file the config refers to, for
// example WithConfigSchemaVariant("discovery-file", "print the schema of a
// discovery file", fn). Only one variant can be asked for at a time.
func WithConfigSchemaVariant(name, usage string, schema func() ([]byte, error)) ConfigCommandOpt {
	return func(c *configCommand) {
		c.schemaVariants = append(c.schemaVariants, schemaVariant{name: name, usage: usage, schema: schema})
	}
}

// WithConfigEnvPrefix makes "config validate" apply the environment variables
// that override the file's settings at startup, as WithEnvOverrides describes,
// so it answers whether the service would start here. appName is the one the
// service passes to InitConfig. The command gets an --ignore-env flag that
// checks the file by itself, for a file whose secrets are only set where it
// is deployed.
//
// Without this option "config validate" checks the file alone.
func WithConfigEnvPrefix(appName string) ConfigCommandOpt {
	return func(c *configCommand) { c.envPrefix = appName }
}

// WithConfigValidateHelp adds text to the end of the help "config validate"
// prints, for what the service's checks do and do not cover.
func WithConfigValidateHelp(text string) ConfigCommandOpt {
	return func(c *configCommand) { c.validateHelp = text }
}

// WithConfigExampleHelp adds text to the end of the help "config example"
// prints.
func WithConfigExampleHelp(text string) ConfigCommandOpt {
	return func(c *configCommand) { c.exampleHelp = text }
}

// WithConfigSchemaHelp adds text to the end of the help "config schema"
// prints.
func WithConfigSchemaHelp(text string) ConfigCommandOpt {
	return func(c *configCommand) { c.schemaHelp = text }
}

// NewConfigCommand returns a "config" command with three subcommands, for a
// service whose config type is built by newConfig and described as kind (for
// example "accounting"):
//
//   - validate checks a config file without starting the service. It reports
//     every key that matches no setting, with its full path, and every setting
//     the config's Validate method rejects, and exits non-zero if there is any.
//     A config that is a ReferenceChecker has what it refers to checked too.
//     It checks the file alone unless WithConfigEnvPrefix is given.
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
	long := fmt.Sprintf(`Check a %[1]s config file without starting the %[1]s.

Reports every key that matches no setting, with its full path (for example a
renamed or misspelled key), and every setting the %[1]s would reject at
startup, all together. Exits non-zero if any problem is found.`, kind)

	cmd := &cobra.Command{
		Use:          "validate",
		Short:        fmt.Sprintf("Check a %s config file for unknown keys and invalid settings", kind),
		Long:         withHelp(long, c.envHelp(kind), c.validateHelp),
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var opts []ValidateConfigFileOpt
			if c.envPrefix != "" {
				ignoreEnv, err := cmd.Flags().GetBool(ignoreEnvFlag)
				if err != nil {
					return err
				}
				if !ignoreEnv {
					opts = append(opts, WithEnvOverrides(c.envPrefix))
				}
			}

			return ValidateConfigFile(cmd.OutOrStdout(), cmd.Flag("config").Value.String(), newConfig(), opts...)
		},
	}
	cmd.Flags().StringP("config", "c", c.defaultFile, "config file to check")
	if c.envPrefix != "" {
		cmd.Flags().Bool(ignoreEnvFlag, false, "check the file by itself, without environment variable overrides")
	}

	return cmd
}

const ignoreEnvFlag = "ignore-env"

// envHelp is the part of the "config validate" help that says whether the
// environment is applied.
func (c *configCommand) envHelp(kind string) string {
	if c.envPrefix == "" {
		return fmt.Sprintf(`Checks the file alone: environment variables that override its settings when
the %s starts are not applied, so a setting the environment supplies is
reported as it is in the file.`, kind)
	}

	return fmt.Sprintf(`Environment variables override the file's settings as they do when the %[1]s
starts, so the result is for this environment: %[2]s overrides
section.some-key. Only a setting the file lists is overridden, so a setting
the environment supplies must be in the file, with an empty value. A variable
that matches no setting is not a problem. Use --%[3]s to check the file by
itself. The result says which was checked.`, kind, envVarName(c.envPrefix, "section.some-key"), ignoreEnvFlag)
}

func (c *configCommand) exampleCommand(kind string) *cobra.Command {
	long := fmt.Sprintf(`Print an example %[1]s config file that lists every setting it can have,
each with its description as a comment and its default value, or an empty
value where it has none.

The example is a reference to copy settings from, not a config to run as it
is: some settings must be set to work.`, kind)

	return &cobra.Command{
		Use:   "example",
		Short: fmt.Sprintf("Print an example %s config file listing every setting", kind),
		Long:  withHelp(long, c.exampleHelp),
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return writeOut(cmd.OutOrStdout(), c.example) },
	}
}

func (c *configCommand) schemaCommand(kind string) *cobra.Command {
	long := fmt.Sprintf(`Print the JSON Schema of a %s config file, for an editor or a validator to
check a file against. The schema checks each key and the type of each value; it
does not apply the rules "config validate" applies beyond that.`, kind)

	cmd := &cobra.Command{
		Use:   "schema",
		Short: fmt.Sprintf("Print the JSON Schema of a %s config file", kind),
		Long:  withHelp(long, c.schemaHelp),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			schema := c.schema
			for _, variant := range c.schemaVariants {
				chosen, err := cmd.Flags().GetBool(variant.name)
				if err != nil {
					return err
				}
				if chosen {
					schema = variant.schema
				}
			}

			return writeOut(cmd.OutOrStdout(), schema)
		},
	}

	names := make([]string, 0, len(c.schemaVariants))
	for _, variant := range c.schemaVariants {
		cmd.Flags().Bool(variant.name, false, variant.usage)
		names = append(names, variant.name)
	}
	if len(names) > 1 {
		cmd.MarkFlagsMutuallyExclusive(names...)
	}

	return cmd
}

// withHelp joins the paragraphs of a command's help, leaving out empty ones.
func withHelp(paragraphs ...string) string {
	var help []string
	for _, p := range paragraphs {
		if p != "" {
			help = append(help, p)
		}
	}

	return strings.Join(help, "\n\n")
}

func writeOut(w io.Writer, generate func() ([]byte, error)) error {
	out, err := generate()
	if err != nil {
		return err
	}
	_, err = w.Write(out)

	return err
}
