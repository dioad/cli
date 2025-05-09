package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/dioad/util"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/dioad/cli/logging"
)

// commandParts
func commandParts(cmd *cobra.Command) []string {
	parts := make([]string, 0)

	if cmd.Parent() != nil {
		parts = append(parts, commandParts(cmd.Parent())...)
	}

	parts = append(parts, cmd.Use)

	return parts
}

func InitViperConfig(orgName, appName string, cfg interface{}) error {
	pflag.Parse()
	return InitViperConfigWithFlagSet(orgName, appName, cfg, pflag.CommandLine)
}

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
			return fmt.Errorf("Fatal error reading config file: %s \n", err)
		}
	}

	err = viper.Unmarshal(cfg)
	if err != nil {
		return err
	}

	return nil
}

func InitConfig(orgName, appName string, cmd *cobra.Command, cfgFile string, cfg interface{}) (*CommonConfig, error) {
	viper.SetEnvPrefix(appName)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	err := viper.BindPFlags(cmd.Flags())
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

	err = viper.Unmarshal(&c) // , viper.DecodeHook(util.MaskedStringDecodeHook))
	if err != nil {
		return nil, err
	}

	logging.ConfigureCmdLogger(c.Logging)

	err = viper.Unmarshal(cfg, viper.DecodeHook(util.MaskedStringDecodeHook))
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

func IsDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	return false
}

func DefaultUserConfigPath(orgName, appName string) (string, error) {
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

func DefaultPersistencePath(orgName, appName string) (string, error) {
	if IsDocker() {
		return "/persist", nil
	}

	return DefaultUserConfigPath(orgName, appName)
}

func DefaultConfigPath(orgName, appName string) (string, error) {
	if IsDocker() {
		return "/config", nil
	}

	return DefaultUserConfigPath(orgName, appName)
}

func DefaultConfigFile(orgName, appName, baseName string) (string, error) {
	userConfigPath, err := DefaultConfigPath(orgName, appName)
	if err != nil {
		return "", err
	}

	return filepath.Join(userConfigPath, fmt.Sprintf("%s.yaml", baseName)), nil
}

func DefaultPersistenceFile(orgName, appName, baseName string) (string, error) {
	userPersistencePath, err := DefaultPersistencePath(orgName, appName)
	if err != nil {
		return "", err
	}

	return filepath.Join(userPersistencePath, fmt.Sprintf("%s.yaml", baseName)), nil
}

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

type configFileContextKey struct{}

type ContextOpt func(context.Context) context.Context

func Context(ctx context.Context, contextOpts ...ContextOpt) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	for _, opt := range contextOpts {
		ctx = opt(ctx)
	}

	return ctx
}

func SetOrgName(orgName string) ContextOpt {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, orgNameContextKey{}, orgName)
	}
}

func SetAppName(appName string) ContextOpt {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, appNameContextKey{}, appName)
	}
}

func getOrgName(ctx context.Context) string {
	orgName, ok := ctx.Value(orgNameContextKey{}).(string)
	if !ok {
		return ""
	}
	return orgName
}

func getAppName(ctx context.Context) string {
	appName, ok := ctx.Value(appNameContextKey{}).(string)
	if !ok {
		return ""
	}
	return appName
}

type CobraOpt[T any] func(*T)

func CobraRunEWithConfig[T any](execFunc func(*T) error, cfg *T) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		orgName := getOrgName(cmd.Context())
		appName := getAppName(cmd.Context())

		var configFile string
		configFlag := cmd.Flag("config")
		if configFlag != nil {
			configFile = configFlag.Value.String()
		}

		_, err := InitConfig(orgName, appName, cmd, configFile, cfg)
		cobra.CheckErr(err)

		return execFunc(cfg)
	}
}

func CobraRunE[T any](execFunc func(*T) error, opt ...CobraOpt[T]) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		orgName := getOrgName(cmd.Context())
		appName := getAppName(cmd.Context())

		var cfg T

		for _, o := range opt {
			o(&cfg)
		}

		_, err := InitConfig(orgName, appName, cmd, "", &cfg)
		cobra.CheckErr(err)

		return execFunc(&cfg)
	}
}
