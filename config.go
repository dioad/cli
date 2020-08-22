package cli

import "C"
import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	dct "github.com/dioadconsulting/net/tls"
)

type TLSConfig struct {

	TLSServerName         string `mapstructure:"tls-server-name"`
	TLSServerCertificate  string `mapstructure:"tls-server-cert"`
	TLSServerKey          string `mapstructure:"tls-server-key"`
	TLSServerCAFile       string `mapstructure:"tls-server-ca-file"`
	TLSServerClientAuth   string `mapstructure:"tls-server-client-auth"`
	TLSServerClientCAFile string `mapstructure:"tls-server-client-ca-file"`

	TLSClientCertificate string `mapstructure:"tls-client-cert"`
	TLSClientKey         string `mapstructure:"tls-client-key"`
	TLSClientSkipVerify  bool   `mapstructure:"tls-client-skip-verify"`
}

func (c TLSConfig) Parse() *tls.Config {
	var tlsConfig = &tls.Config{}
	if C.TLSServerCertificate != "" {
		serverCertificate, err := dct.LoadServerCertificateFromConfig(C.TLSConfig)

		if err != nil {
			log.Fatal(err)
		}
		tlsConfig.Certificates = []tls.Certificate{*serverCertificate}
	}

	return tlsConfig
}

func InitViperConfig(orgName, appName string) error {
	pflag.Parse()

	viper.BindPFlags(pflag.CommandLine)

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

	return nil
}