package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/dioad/cli/logging"
	"github.com/dioad/config/configcheck"
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
// Configuration sources are merged with Viper's standard precedence (highest to
// lowest): explicit overrides, command-line flags, environment variables
// (prefixed with appName), config files, then defaults.
//
// Deprecated: use InitConfig instead. InitViperConfig and
// InitViperConfigWithFlagSet operate on Viper's global package-level singleton,
// so concurrent or repeated calls within the same process (including parallel
// tests) mutate shared state. InitConfig instead constructs a local Viper
// instance per call and is safe to use from parallel tests or multiple
// commands in the same process.
func InitViperConfig(orgName, appName string, cfg any) error {
	pflag.Parse()
	return InitViperConfigWithFlagSet(orgName, appName, cfg, pflag.CommandLine)
}

// InitViperConfigWithFlagSet initializes Viper with a custom FlagSet.
//
// Similar to InitViperConfig but allows specifying a custom pflag.FlagSet
// instead of using the global command line flags. Useful for embedding
// configuration initialization in library code or tests.
//
// Deprecated: use InitConfig instead, for the same reason as InitViperConfig —
// this function reads and mutates Viper's global singleton. Note also that its
// environment variable key replacer only maps "-" to "_", whereas InitConfig's
// also maps "." to "_"; the two functions are not behaviourally interchangeable
// for keys containing dots.
func InitViperConfigWithFlagSet(orgName, appName string, cfg any, parsedFlagSet *pflag.FlagSet) error {
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
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			return fmt.Errorf("fatal error reading config file: %w", err)
		}
	}

	err = viper.Unmarshal(cfg)
	if err != nil {
		return fmt.Errorf("error unmarshalling config: %w", err)
	}

	return nil
}

// InitConfigOption is a functional option for customising InitConfig behaviour.
type InitConfigOption func(*initConfigOptions)

type initConfigOptions struct {
	configureLogging bool
	watchConfig      bool
}

func defaultInitConfigOptions() initConfigOptions {
	return initConfigOptions{
		configureLogging: true,
		watchConfig:      true,
	}
}

// WithoutLogging disables automatic logging configuration during InitConfig.
//
// By default, InitConfig configures the global logger via ConfigureProcessLogging.
// Use this option in tests or when managing logging independently.
func WithoutLogging() InitConfigOption {
	return func(o *initConfigOptions) {
		o.configureLogging = false
	}
}

// WithoutWatchConfig is a no-op, retained for backward compatibility.
//
// Deprecated: InitConfig no longer starts a background config-file watcher.
// The previous implementation started an fsnotify goroutine via Viper's
// WatchConfig but never re-applied changes into the caller's config struct,
// and doing so safely would require synchronized access to that struct (a
// breaking API change). The goroutine was therefore removed rather than
// left running with no observable effect.
func WithoutWatchConfig() InitConfigOption {
	return func(o *initConfigOptions) {
		o.watchConfig = false
	}
}

// InitConfig loads and initializes configuration from multiple sources.
//
// It integrates Cobra commands with Viper configuration management, supporting:
// - Hierarchical command-based config file naming
// - Flag binding from the Cobra command
// - Environment variable overrides
// - Automatic logging configuration (disable with WithoutLogging)
func InitConfig(orgName, appName string, cmd *cobra.Command, cfg any, opts ...InitConfigOption) (*CommonConfig, error) {
	options := defaultInitConfigOptions()
	for _, o := range opts {
		o(&options)
	}

	err := ValidateOrgAndAppName(orgName, appName)
	if err != nil {
		return nil, fmt.Errorf("error validating org and app name: %w", err)
	}

	v := viper.New()

	v.SetEnvPrefix(appName)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	err = bindFlags(v, cmd.Flags())
	if err != nil {
		return nil, err
	}

	configureConfigFile(v, orgName, appName, cmd)

	v.AutomaticEnv()

	err = readConfigFile(v)
	if err != nil {
		return nil, err
	}

	var c CommonConfig

	err = UnmarshalConfig(v, &c)
	if err != nil {
		return nil, err
	}

	if options.configureLogging {
		logging.ConfigureProcessLogging(c.Logging)
	}

	err = UnmarshalConfig(v, cfg)
	if err != nil {
		return &c, err
	}

	return &c, nil
}

// configureConfigFile points v at the config file InitConfig reads for cmd:
// the --config flag's value when it is set, otherwise a file named after
// the command path in the system and user config directories.
func configureConfigFile(v *viper.Viper, orgName, appName string, cmd *cobra.Command) {
	var cfgFile string
	if configFlag := cmd.Flag("config"); configFlag != nil {
		cfgFile = configFlag.Value.String()
	}

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
		return
	}

	v.AddConfigPath(filepath.Join("/etc", orgName, appName))

	homeConfigPath := "/config"
	if home, err := homedir.Dir(); err == nil {
		homeConfigPath = filepath.Join(home, ".config", orgName, appName)
	}
	v.AddConfigPath(homeConfigPath)

	v.SetConfigName(strings.Join(commandParts(cmd), "-"))
	v.SetConfigType("yaml")
}

// readConfigFile reads the file v is configured for. Not finding a file on
// the search path is not an error.
func readConfigFile(v *viper.Viper) error {
	err := v.ReadInConfig()
	if err == nil {
		return nil
	}
	if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
		return nil
	}
	return fmt.Errorf("fatal error reading config file: %w", err)
}

// UnmarshalConfig unmarshals Viper configuration into the provided struct with custom decode hooks.
//
// It uses a composed decode hook to handle special types like MaskedString, time.Duration,
// net.IP, and net.IPNet. This allows for seamless unmarshalling of complex configuration fields.
//
// The hook is configcheck.DecodeHook, so configcheck.Decode accepts exactly
// the values this function does.
//
// Viper answers for every bound flag, set or not, so left alone an unset slice
// flag decodes as an empty slice, and every pointer field gets allocated. Neither
// equals its zero value, so a check for whether a field is set would see it
// as set. UnmarshalConfig decodes an empty slice as nil, and leaves a pointer
// field nil unless the config file, the environment or a flag on the command
// line sets it or something under it.
func UnmarshalConfig(v *viper.Viper, c any) error {
	settings := v.AllSettings()
	dropUnsetPointers(v, settings, reflect.TypeOf(c), nil)

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(configcheck.DecodeHook(), dropEmptySlice),
		WeaklyTypedInput: true,
		Result:           c,
	})
	if err != nil {
		return err
	}

	return decoder.Decode(settings)
}

// dropEmptySlice decodes an empty slice as the zero value of its target. It is
// composed last because a hook that returns nil leaves the ones after it
// nothing to convert.
func dropEmptySlice(from, to reflect.Value) (any, error) {
	if from.Kind() == reflect.Slice && to.Kind() == reflect.Slice && from.Len() == 0 {
		return nil, nil
	}

	return from.Interface(), nil
}

// dropUnsetPointers removes from settings the entries for the pointer fields of
// t that nothing but a flag's default gives a value, descending into the
// fields it keeps. path is the key of settings within the config.
func dropUnsetPointers(v *viper.Viper, settings map[string]any, t reflect.Type, path []string) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	for field := range t.Fields() {
		name, options, _ := strings.Cut(field.Tag.Get("mapstructure"), ",")
		squash := field.Anonymous && name == "" || strings.Contains(options, "squash")
		if squash {
			dropUnsetPointers(v, settings, field.Type, path)

			continue
		}
		if name == "" {
			name = field.Name
		}
		name = strings.ToLower(name)

		fieldPath := append(path[:len(path):len(path)], name)
		if field.Type.Kind() == reflect.Pointer && !isConfigured(v, strings.Join(fieldPath, ".")) {
			delete(settings, name)

			continue
		}
		if nested, ok := settings[name].(map[string]any); ok {
			dropUnsetPointers(v, nested, field.Type, fieldPath)
		}
	}
}

// isConfigured reports whether key, or a key under it, has a value from the
// config file, the environment or a flag that was set. A flag's default does
// not count.
func isConfigured(v *viper.Viper, key string) bool {
	for _, k := range v.AllKeys() {
		if (k == key || strings.HasPrefix(k, key+".")) && v.IsSet(k) {
			return true
		}
	}

	return false
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

// CobraRunE returns a Cobra RunE function with configuration management and functional options.
//
// The returned function handles configuration initialization with org and app names from context,
// applies functional options to modify configuration, and passes configured values to the execution function.
func CobraRunE[T any](execFunc func(context.Context, *T) error, opt ...CobraOpt[T]) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		orgName := OrgNameFromContext(cmd.Context())
		appName := AppNameFromContext(cmd.Context())

		if orgName == "" || appName == "" {
			return fmt.Errorf("cli: org name and app name must be set in context via SetOrgName/SetAppName")
		}

		var cfg T

		for _, o := range opt {
			o(&cfg)
		}

		runOpts := runOptionsFromContext(cmd.Context())

		var configFile string
		var fileErr error
		if runOpts.checkFile != nil {
			configFile, fileErr = checkConfigFile(orgName, appName, cmd, runOpts.checkFile)
		}

		_, err := InitConfig(orgName, appName, cmd, &cfg)
		if err != nil {
			// The strict check decodes the same file, so its findings
			// already name the fields InitConfig failed on, with paths.
			if fileErr != nil {
				return invalidConfigError(configFile, fileErr)
			}
			return err
		}

		if err := validateConfig(runOpts, fileErr, &cfg); err != nil {
			return invalidConfigError(configFile, err)
		}

		// Attach the logger InitConfig just configured to the context so
		// execFunc can retrieve it via zerolog.Ctx(ctx) instead of depending
		// on the global logger directly.
		ctx := log.Logger.WithContext(cmd.Context())

		return execFunc(ctx, &cfg)
	}
}

// bindFlags binds fs's flags into v. A flag generated from a config struct
// field binds to the config key the field is decoded from, not to its own name,
// so its value reaches the config; any other flag binds under its name. A
// deprecated alias that was set counts as its canonical flag being set.
func bindFlags(v *viper.Viper, fs *pflag.FlagSet) error {
	var err error
	fs.VisitAll(func(f *pflag.Flag) {
		if err != nil {
			return
		}
		if target := annotation(f, aliasOfAnnotation); target != "" {
			if canonical := fs.Lookup(target); canonical != nil && f.Changed {
				canonical.Changed = true
			}

			return
		}

		key := annotation(f, configKeyAnnotation)
		if key == "" {
			key = f.Name
		}
		err = v.BindPFlag(key, f)
	})

	return err
}

func annotation(f *pflag.Flag, name string) string {
	if values := f.Annotations[name]; len(values) > 0 {
		return values[0]
	}

	return ""
}
