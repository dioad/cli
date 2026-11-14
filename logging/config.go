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
	Level string `desc:"log level: trace, debug, info, warn, error, fatal or panic" mapstructure:"level"`

	// File specifies the path to the log file. Supports home directory expansion (~).
	// Empty string disables file logging (logs to stdout only).
	File string `desc:"log file path, with ~ expanded; empty logs to stdout only" mapstructure:"file"`

	// MaxSize is the maximum size of the log file in MB before rotation occurs.
	// Default is 100 MB.
	MaxSize int `desc:"maximum log file size in MB before it is rotated" mapstructure:"max-size"`

	// MaxAge is the maximum number of days to keep old log files.
	// Default is 0 (no limit).
	MaxAge int `desc:"days to keep rotated log files; 0 keeps them all" mapstructure:"max-age"`

	// MaxBackups is the maximum number of old log files to retain.
	// Default is 0 (keep all).
	MaxBackups int `desc:"rotated log files to keep; 0 keeps them all" mapstructure:"max-backups"`

	// LocalTime determines if rotated filenames use local time instead of UTC.
	// Default is false.
	LocalTime bool `desc:"name rotated log files with local time instead of UTC" mapstructure:"use-local-time"`

	// Compress determines if old log files are gzip-compressed after rotation.
	// Default is false.
	Compress bool `desc:"gzip rotated log files" mapstructure:"compress"`

	// Mode is reserved for future use.
	Mode string `flag:",hidden" mapstructure:"mode"`
}
