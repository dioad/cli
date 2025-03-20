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
	DefaultLogLevel = zerolog.WarnLevel
)

type Option func(*Config)

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

func ConfigureCmdLogger(c Config, opts ...Option) {
	for _, o := range opts {
		o(&c)
	}

	ConfigureLogLevel(c.Level, DefaultLogLevel)
	ConfigureLogOutput(c)
}

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

func FatalError(err error) {
	log.Fatal().Err(err)
}
