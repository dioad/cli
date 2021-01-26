package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func InitViperConfig(orgName, appName string, cfg interface{}) error {
	pflag.Parse()
	return InitViperConfigWithFlagSet(orgName, appName, cfg, pflag.CommandLine)
}

func InitViperConfigWithFlagSet(orgName, appName string, cfg interface{}, parsedFlagSet *pflag.FlagSet) error {

	viper.BindPFlags(parsedFlagSet)

	viper.SetConfigName(appName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(fmt.Sprintf("/etc/%s/%s", orgName, appName))
	viper.AddConfigPath(fmt.Sprintf("$HOME/.%s/%s", orgName, appName))
	viper.AddConfigPath(".")
	viper.SetEnvPrefix(appName)
	if viper.GetString("config") != "" {
		viper.SetConfigFile(viper.GetString("config"))
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	err := viper.ReadInConfig()
	if err != nil { // Handle errors reading the config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("Fatal error reading config file: %s \n", err)
		}
	}

	err = viper.Unmarshal(cfg)
	if err != nil {
		return err
	}

	return nil
}
