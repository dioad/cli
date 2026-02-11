package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/dioad/util"

	"github.com/dioad/cli/logging"
)

// commandParts reconstructs the full command path by traversing parent commands.
// Used internally to build hierarchical command names for configuration lookup.
func commandParts(cmd *cobra.Command) []string {
	parts := make([]string, 0)

	if cmd.Parent() != nil {
		parts = append(parts, commandParts(cmd.Parent())...)
	}

	parts = append(parts, cmd.Use)

	return parts
}

// InitViperConfig initializes Viper configuration management with parsed command-line flags.
//
// It sets up Viper to:
// - Bind command-line flags
// - Search for configuration files in standard locations
// - Support environment variables with the given appName prefix
// - Unmarshal configuration into the provided cfg struct
//
// Configuration sources are merged with this precedence (highest to lowest):
// 1. Command-line flags
// 2. Explicit config file (--config flag)
// 3. Environment variables (prefixed with appName)
// 4. Config files in standard locations
func InitViperConfig(orgName, appName string, cfg interface{}) error {
	pflag.Parse()
	return InitViperConfigWithFlagSet(orgName, appName, cfg, pflag.CommandLine)
}

// InitViperConfigWithFlagSet initializes Viper with a custom FlagSet.
//
// Similar to InitViperConfig but allows specifying a custom pflag.FlagSet
// instead of using the global command line flags. Useful for embedding
// configuration initialization in library code or tests.
func InitViperConfigWithFlagSet(orgName, appName string, cfg interface{}, parsedFlagSet *pflag.FlagSet) error {
	err := viper.BindPFlags(parsedFlagSet)
	if err != nil {
		return fmt.Errorf("error binding persistent flags: %w", err)
	}

	viper.SetConfigName(appName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(fmt.Sprintf("/etc/%s/%s", orgName, appName))
	viper.AddConfigPath(fmt.Sprintf("$HOME/.%s/%s", orgName, appName))
	viper.AddConfigPath(fmt.Sprintf("$HOME/.config/%s/%s", orgName, appName))
	viper.AddConfigPath(".")
	viper.SetEnvPrefix(appName)
	if viper.GetString("config") != "" {
		viper.SetConfigFile(viper.GetString("config"))
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	err = viper.ReadInConfig()
	if err != nil { // Handle errors reading the config file
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("fatal error reading config file: %s", err)
		}
	}

	err = viper.Unmarshal(cfg)
	if err != nil {
		return err
	}

	return nil
}

// InitConfig loads and initializes configuration from multiple sources.
//
// It integrates Cobra commands with Viper configuration management, supporting:
// - Hierarchical command-based config file naming
// - Flag binding from the Cobra command
// - Environment variable overrides
// - Automatic logging configuration
// - Configuration hot-reloading via Viper watchers
func InitConfig(orgName, appName string, cmd *cobra.Command, cfgFile string, cfg interface{}) (*CommonConfig, error) {
	err := ValidateOrgAndAppName(orgName, appName)
	if err != nil {
		return nil, fmt.Errorf("error validating org and app name: %w", err)
	}

	viper.SetEnvPrefix(appName)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	err = viper.BindPFlags(cmd.Flags())
	if err != nil {
		return nil, err
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		systemConfigPath := filepath.Join("/etc", orgName, appName)

		homeConfigPath := "/config"
		// Find home directory.
		home, err := homedir.Dir()
		if err == nil {
			homeConfigPath = filepath.Join(home, ".config", orgName, appName)
		}

		fullCommandName := fmt.Sprintf("%v\n", strings.Join(commandParts(cmd), "-"))

		// Search config in home directory with name "config" (without extension).
		viper.SetConfigName(fullCommandName)
		viper.SetConfigType("yaml")
		viper.AddConfigPath(systemConfigPath)
		viper.AddConfigPath(homeConfigPath)
	}
	viper.AutomaticEnv()
	// If a config file is found, read it in.
	err = viper.ReadInConfig()

	// cobra.CheckErr(err)

	// viper.AutomaticEnv() // read in environment variables that match
	if err == nil {
		// Commenting this bit out as I don't like the error
		// when running commands like `version`
		//	log.Trace().Err(err).
		//		Str("file", viper.ConfigFileUsed()).
		//		Msg("error reading config")
		// } else {
		viper.WatchConfig()
	}

	var c CommonConfig

	err = viper.Unmarshal(&c)
	if err != nil {
		return nil, err
	}

	logging.ConfigureCmdLogger(c.Logging)

	err = UnmarshalConfig(cfg)
	if err != nil {
		return &c, err
	}

	log.Logger.Debug().
		Str("file", viper.ConfigFileUsed()).
		Interface("config", cfg).
		Interface("common", c).
		Msg("initialising")

	return &c, nil
}

// UnmarshalConfig unmarshals Viper configuration into the provided struct with custom decode hooks.
//
// It uses a composed decode hook to handle special types like MaskedString, time.Duration,
// net.IP, and net.IPNet. This allows for seamless unmarshalling of complex configuration fields.
func UnmarshalConfig(c interface{}) error {
	decodeHook := mapstructure.ComposeDecodeHookFunc(
		util.MaskedStringDecodeHook,
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToIPHookFunc(),
		mapstructure.StringToIPNetHookFunc(),
	)
	return viper.Unmarshal(c, viper.DecodeHook(decodeHook))
}

// IsDocker detects if the application is running inside a Docker container.
//
// It checks for the presence of /.dockerenv file, which is a standard
// indicator that the process is running in a Docker container.
func IsDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	return false
}

// ValidateName checks that the provided name is valid for use in configuration paths.
//
// It ensures that the name is not empty, does not contain path separators,
// does not start or end with spaces, and does not contain special characters.
// This validation helps prevent directory traversal issues and ensures clean config paths.
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name must not be empty")
	}

	if strings.Contains(name, string(os.PathSeparator)) {
		return fmt.Errorf("name must not contain path separators")
	}

	if name == "." || name == ".." {
		return fmt.Errorf("name must not be '.' or '..'")
	}
	if strings.HasPrefix(name, " ") {
		return fmt.Errorf("name must not start with a space")
	}

	if strings.HasSuffix(name, " ") {
		return fmt.Errorf("name must not end with a space")
	}

	if strings.Contains(name, " ") {
		return fmt.Errorf("name must not contain spaces")
	}

	specialChars := "/\\:*?\"<>|(){}[]!@#$%^&*+=~`"
	for _, char := range specialChars {
		if strings.Contains(name, string(char)) {
			return fmt.Errorf("name must not contain special characters like %s", string(char))
		}
	}

	return nil
}

// ValidateOrgAndAppName validates that orgName and appName are acceptable names.
//
// It delegates to ValidateName, which ensures that names are non-empty, do not
// contain path separators, spaces (including leading or trailing spaces), or
// various special characters. This helps prevent issues when constructing
// configuration paths and other filesystem-related operations.
func ValidateOrgAndAppName(orgName, appName string) error {
	if err := ValidateName(orgName); err != nil {
		return fmt.Errorf("invalid orgName: %w", err)
	}

	if err := ValidateName(appName); err != nil {
		return fmt.Errorf("invalid appName: %w", err)
	}

	return nil
}

// DefaultUserConfigPath returns the default configuration directory for the user.
//
// For non-Docker environments, it returns $HOME/.config/{orgName}/{appName}
// and creates the directory if it doesn't exist with 0700 permissions.
func DefaultUserConfigPath(orgName, appName string) (string, error) {
	err := ValidateOrgAndAppName(orgName, appName)
	if err != nil {
		return "", fmt.Errorf("error validating org and app name: %w", err)
	}

	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	userConfigPath := filepath.Join(currentUser.HomeDir, ".config", orgName, appName)
	err = os.MkdirAll(userConfigPath, 0700)
	if err != nil {
		return "", err
	}
	return userConfigPath, nil
}

// DefaultPersistencePath returns the default directory for persistent application data.
//
// For Docker containers, this returns /persist.
// For other environments, it returns the same as DefaultUserConfigPath.
func DefaultPersistencePath(orgName, appName string) (string, error) {
	if IsDocker() {
		return "/persist", nil
	}

	err := ValidateOrgAndAppName(orgName, appName)
	if err != nil {
		return "", fmt.Errorf("error validating org and app name: %w", err)
	}

	return DefaultUserConfigPath(orgName, appName)
}

// DefaultConfigPath returns the default directory for configuration files.
//
// For Docker containers, this returns /config.
// For other environments, it returns $HOME/.config/{orgName}/{appName}.
func DefaultConfigPath(orgName, appName string) (string, error) {
	if IsDocker() {
		return "/config", nil
	}

	return DefaultUserConfigPath(orgName, appName)
}

// DefaultConfigFile returns the full path to the default configuration file.
//
// The file is placed in DefaultConfigPath and named {baseName}.yaml.
func DefaultConfigFile(orgName, appName, baseName string) (string, error) {
	userConfigPath, err := DefaultConfigPath(orgName, appName)
	if err != nil {
		return "", err
	}

	return filepath.Join(userConfigPath, fmt.Sprintf("%s.yaml", baseName)), nil
}

// DefaultPersistenceFile returns the full path to a persistence file.
//
// The file is placed in DefaultPersistencePath and named {baseName}.yaml.
func DefaultPersistenceFile(orgName, appName, baseName string) (string, error) {
	userPersistencePath, err := DefaultPersistencePath(orgName, appName)
	if err != nil {
		return "", err
	}

	return filepath.Join(userPersistencePath, fmt.Sprintf("%s.yaml", baseName)), nil
}

// CommonConfig contains configuration shared across all applications.
//
// It includes logging configuration and can be extended in application-specific
// config structs via embedding.
type CommonConfig struct {
	// Config  string         `mapstructure:"config"`
	Logging logging.Config `mapstructure:"log"`
}

type Config[T any] struct {
	CommonConfig
	Config *T
}

type orgNameContextKey struct{}
type appNameContextKey struct{}

// ContextOpt is a functional option for building a context with application metadata.
type ContextOpt func(context.Context) context.Context

// Context creates a new context with optional application metadata.
//
// It accepts functional options to populate the context with organization and app names.
func Context(ctx context.Context, contextOpts ...ContextOpt) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	for _, opt := range contextOpts {
		ctx = opt(ctx)
	}

	return ctx
}

// SetOrgName returns a ContextOpt that stores the organization name in the context.
func SetOrgName(orgName string) ContextOpt {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, orgNameContextKey{}, orgName)
	}
}

// SetAppName returns a ContextOpt that stores the application name in the context.
func SetAppName(appName string) ContextOpt {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, appNameContextKey{}, appName)
	}
}

// OrgNameFromContext retrieves the organization name from the context.
func OrgNameFromContext(ctx context.Context) string {
	orgName, ok := ctx.Value(orgNameContextKey{}).(string)
	if !ok {
		return ""
	}
	return orgName
}

// AppNameFromContext retrieves the application name from the context.
func AppNameFromContext(ctx context.Context) string {
	appName, ok := ctx.Value(appNameContextKey{}).(string)
	if !ok {
		return ""
	}
	return appName
}

// CobraOpt is a functional option for configuring command execution.
type CobraOpt[T any] func(*T)

// CobraRunEWithConfig returns a Cobra RunE function that loads configuration before execution.
//
// The returned function handles configuration loading, application metadata retrieval,
// and passes configured values to the execution function.
func CobraRunEWithConfig[T any](execFunc func(context.Context, *T) error, cfg *T) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		orgName := OrgNameFromContext(cmd.Context())
		appName := AppNameFromContext(cmd.Context())

		var configFile string
		configFlag := cmd.Flag("config")
		if configFlag != nil {
			configFile = configFlag.Value.String()
		}
		_, err := InitConfig(orgName, appName, cmd, configFile, cfg)
		cobra.CheckErr(err)

		return execFunc(cmd.Context(), cfg)
	}
}

// CobraRunE returns a Cobra RunE function with configuration management and functional options.
//
// The returned function handles configuration initialization with org and app names from context,
// applies functional options to modify configuration, and passes configured values to the execution function.
func CobraRunE[T any](execFunc func(*T) error, opt ...CobraOpt[T]) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		orgName := OrgNameFromContext(cmd.Context())
		appName := AppNameFromContext(cmd.Context())

		var cfg T

		for _, o := range opt {
			o(&cfg)
		}

		_, err := InitConfig(orgName, appName, cmd, "", &cfg)
		cobra.CheckErr(err)

		return execFunc(&cfg)
	}
}
