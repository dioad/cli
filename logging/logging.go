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

//func init() {
//	zerolog.TimeFieldFormat = time.RFC3339Nano
//
//	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
//		output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339Nano}
//		log.Logger = zerolog.New(output).With().Timestamp().Logger()
//	} else {
//		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
//	}
//}

func ConfigureCmdLogger(c Config) {
	ConfigureLogLevel(c.Level)
	ConfigureLogOutput(c)
}

func ConfigureLogLevel(levelString string) {
	// Configure logging
	zerolog.TimeFieldFormat = time.RFC3339Nano

	logLevel, err := zerolog.ParseLevel(levelString)
	if err != nil {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Warn().Err(err).Msg("failed to parse log level. Defaulting to INFO")
	} else if logLevel.String() != "" {
		zerolog.SetGlobalLevel(logLevel)
	}
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
