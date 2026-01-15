package logging

// Config contains all configuration options for logging behavior.
//
// Fields support struct tags for configuration file unmarshaling:
//
//	type MyConfig struct {
//		Logging Config `mapstructure:"log"`
//	}
//
// YAML Example:
//
//	log:
//	  level: debug
//	  file: "~/.app/logs/app.log"
//	  max-size: 100
//	  max-backups: 3
//	  max-age: 7
//	  use-local-time: false
//	  compress: true
type Config struct {
	// Level specifies the logging level: trace, debug, info, warn, error, fatal, panic.
	// Empty string defaults to application-configured level or zerolog.WarnLevel.
	Level string `mapstructure:"level"`

	// File specifies the path to the log file. Supports home directory expansion (~).
	// Empty string disables file logging (logs to stdout only).
	File string `mapstructure:"file"`

	// MaxSize is the maximum size of the log file in MB before rotation occurs.
	// Default is 100 MB.
	MaxSize int `mapstructure:"max-size"`

	// MaxAge is the maximum number of days to keep old log files.
	// Default is 0 (no limit).
	MaxAge int `mapstructure:"max-age"`

	// MaxBackups is the maximum number of old log files to retain.
	// Default is 0 (keep all).
	MaxBackups int `mapstructure:"max-backups"`

	// LocalTime determines if rotated filenames use local time instead of UTC.
	// Default is false.
	LocalTime bool `mapstructure:"use-local-time"`

	// Compress determines if old log files are gzip-compressed after rotation.
	// Default is false.
	Compress bool `mapstructure:"compress"`

	// Mode is reserved for future use.
	Mode string `mapstructure:"mode"`
}
