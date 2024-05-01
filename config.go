package cli

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/dioad/cli/logging"
)

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

func InitConfig(orgName, appName, cmdName, cfgFile string, cfg interface{}) (*CommonConfig, error) {
	viper.SetEnvPrefix(appName)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	err := viper.BindPFlags(pflag.CommandLine)
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

		// Search config in home directory with name "config" (without extension).
		viper.SetConfigName(cmdName)
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

	err = viper.Unmarshal(cfg)
	if err != nil {
		return &c, err
	}

	log.Debug().
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
	Logging logging.Config `mapstructure:"log"`
}
