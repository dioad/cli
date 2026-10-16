package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dioad/config/configcheck"
)

// Validator is implemented by a configuration type that can check itself.
// With WithConfigValidation, CobraRunE calls Validate on the loaded
// configuration before the command's run function.
type Validator interface {
	Validate() error
}

// runOptions are the CobraRunE settings made by CommandOpts. They travel to
// CobraRunE on the command's context because CommandOpt can only change
// the command, not the RunE closure NewCommand builds.
type runOptions struct {
	// checkFile strictly decodes the config file's settings into the
	// file's own type.
	checkFile func(settings map[string]any) error
	validate  bool
}

type runOptionsContextKey struct{}

func runOptionsFromContext(ctx context.Context) runOptions {
	if ctx == nil {
		return runOptions{}
	}
	opts, _ := ctx.Value(runOptionsContextKey{}).(runOptions)
	return opts
}

// withRunOption wraps cmd.RunE so set is applied to the runOptions
// CobraRunE reads. It must run after RunE is set, as NewCommand's options
// do.
func withRunOption(set func(*runOptions)) CommandOpt {
	return func(cmd *cobra.Command) {
		next := cmd.RunE
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			if next == nil {
				return fmt.Errorf("cli: command %q has no RunE; pass this option to NewCommand", cmd.Name())
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			opts := runOptionsFromContext(ctx)
			set(&opts)
			cmd.SetContext(context.WithValue(ctx, runOptionsContextKey{}, opts))
			return next(cmd, args)
		}
	}
}

// WithStrictConfigFile makes the command reject a config file containing
// keys that match no field of F, reporting each as a configcheck finding
// with its full path.
//
// F is the type of the whole config file, which need not be the command's
// own config type: a subcommand that reads only part of a shared file
// passes the file's full type, so it rejects the same typos the main
// command does without rejecting keys it simply does not use. Only the file
// is checked; flags and environment variables are not.
//
// A key whose value is null or an empty map is not reported: viper drops
// such keys when it reads the file, so they carry no settings to lose.
func WithStrictConfigFile[F any]() CommandOpt {
	return withRunOption(func(o *runOptions) {
		o.checkFile = func(settings map[string]any) error {
			return configcheck.Decode(settings, new(F))
		}
	})
}

// WithConfigValidation makes CobraRunE call Validate on the loaded
// configuration, when its type implements Validator, before the command's
// run function. Its findings are reported together with any from
// WithStrictConfigFile.
func WithConfigValidation() CommandOpt {
	return withRunOption(func(o *runOptions) {
		o.validate = true
	})
}

// DecodeConfigFile strictly decodes the config file at path into out, using
// the same decode hook as InitConfig and rejecting keys that match no field.
// Problems with the file's contents are returned as a *configcheck.Error.
// Flags and environment variables are not applied. As with
// WithStrictConfigFile, keys with a null or empty-map value are not
// reported.
func DecodeConfigFile(path string, out any) error {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}
	return configcheck.Decode(v.AllSettings(), out)
}

// checkConfigFile runs check over the settings of the config file
// InitConfig would read for cmd, returning the file's path and check's
// result. It returns no error when the file cannot be read, leaving
// InitConfig to report that.
func checkConfigFile(orgName, appName string, cmd *cobra.Command, check func(map[string]any) error) (string, error) {
	v := viper.New()
	configureConfigFile(v, orgName, appName, cmd)
	if err := v.ReadInConfig(); err != nil {
		return "", nil //nolint:nilerr // InitConfig reads the same file and reports the error.
	}
	return v.ConfigFileUsed(), check(v.AllSettings())
}

// invalidConfigError reports err, which lists one finding per line, as an
// invalid configuration. errors.As still reaches the *configcheck.Error.
func invalidConfigError(configFile string, err error) error {
	if configFile == "" {
		return fmt.Errorf("invalid configuration:\n%w", err)
	}
	return fmt.Errorf("invalid configuration in %s:\n%w", configFile, err)
}

// validateConfig gathers the strict file check's findings and, when
// enabled, cfg's own validation into one error.
func validateConfig(opts runOptions, fileErr error, cfg any) error {
	c := configcheck.NewCollector()
	c.Add(fileErr)
	if v, ok := cfg.(Validator); ok && opts.validate {
		c.Add(v.Validate())
	}
	return c.Err()
}
