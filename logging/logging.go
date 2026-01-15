package logging

import (
	"io"
	defaultLog "log"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// DefaultLogLevel is the default logging level when none is configured.
	DefaultLogLevel = zerolog.WarnLevel
)

// Option is a functional option for configuring logging behavior.
type Option func(*Config)

// WithDefaultLogLevel returns an Option that sets a default log level if one isn't already configured.
//
// This option is useful for providing application-level defaults that won't override
// explicit configuration from config files or environment variables.
func WithDefaultLogLevel(level zerolog.Level) func(*Config) {
	return func(c *Config) {
		if c.Level == "" {
			c.Level = level.String()
			return
		}

		_, err := zerolog.ParseLevel(c.Level)
		if err != nil {
			c.Level = level.String()
			return
		}
	}
}

// ConfigureCmdLogger configures the global zerolog logger with the provided settings and options.
//
// It applies all options, then sets up log level and output configuration.
// This is the main entry point for logging setup in CLI applications.
func ConfigureCmdLogger(c Config, opts ...Option) {
	for _, o := range opts {
		o(&c)
	}

	ConfigureLogLevel(c.Level, DefaultLogLevel)
	ConfigureLogOutput(c)
}

// ConfigureLogLevel sets the global log level for zerolog.
//
// If levelString is empty or invalid, defaultLogLevel is used.
// Valid levels: trace, debug, info, warn, error, fatal, panic.
func ConfigureLogLevel(levelString string, defaultLogLevel zerolog.Level) {
	// Configure logging
	zerolog.TimeFieldFormat = time.RFC3339Nano

	if levelString == "" {
		zerolog.SetGlobalLevel(defaultLogLevel)
		return
	}

	logLevel, err := zerolog.ParseLevel(levelString)
	if err != nil {
		zerolog.SetGlobalLevel(defaultLogLevel)
		log.Warn().Err(err).Msg("failed to parse log level. Defaulting to INFO")
		return
	}

	zerolog.SetGlobalLevel(logLevel)

}

func isConsoleWriter(f *os.File) bool {
	fileInfo, _ := f.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ConfigureLogFileOutput creates and returns a rotated file writer using lumberjack.
//
// The provided Config struct must specify the File path and optional rotation settings.
// File paths support home directory expansion (e.g., "~/.app/logs/app.log").
func ConfigureLogFileOutput(c Config) io.Writer {
	expandedDir, err := homedir.Expand(c.File)
	if err != nil {
		log.Error().
			Str("filePath", c.File).
			Err(err).
			Msg("unable to expand log file path")
		return nil
	}

	filePath := filepath.Clean(expandedDir)

	logOutput := &lumberjack.Logger{
		Filename:   filepath.Clean(filePath),
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		LocalTime:  c.LocalTime,
		Compress:   c.Compress,
	}

	return logOutput
}

// ConfigureLogOutput sets up the global zerolog logger output.
//
// If stdout is a TTY (terminal), output is formatted as pretty-printed JSON.
// If a log file is configured, output is directed to the file with rotation.
// Otherwise, output is directed to stdout as compact JSON.
func ConfigureLogOutput(c Config) {

	var logOutput io.Writer
	logOutput = os.Stdout

	// Setup logging to stdout by default
	// so we have somewhere to log any errors configuring logging
	if isConsoleWriter(os.Stdout) {
		logOutput = zerolog.ConsoleWriter{Out: logOutput, TimeFormat: time.RFC3339Nano}
	}
	log.Logger = zerolog.New(logOutput).With().Timestamp().Logger()

	// if a log file has been configured set it up and
	// overwrite default logger
	if c.File != "" {
		logOutput = ConfigureLogFileOutput(c)
	}
	log.Logger = zerolog.New(logOutput).With().Timestamp().Logger()

	// Configure default logger
	defaultLog.SetFlags(0 | defaultLog.Lshortfile)
	defaultLog.SetOutput(log.Logger.With().Str("level", "default").Logger())
}

// FatalError logs an error with fatal level and exits the program.
func FatalError(err error) {
	log.Fatal().Err(err)
}
