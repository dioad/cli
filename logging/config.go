package logging

type Config struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max-size"`
	MaxAge     int    `mapstructure:"max-age"`
	MaxBackups int    `mapstructure:"max-backups"`
	LocalTime  bool   `mapstructure:"use-local-time"`
	Compress   bool   `mapstructure:"compress"`
	Mode       string `mapstructure:"mode"`
}
