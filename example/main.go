package main

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/dioad/cli/logging"
)

func testLogLevel(logLevel string) {
	logging.ConfigureLogLevel(logLevel, zerolog.NoLevel)

	fmt.Printf("Testing Log level \"%s\"\n", logLevel)

	log.Trace().Msgf("Log level set to %s", logLevel)
	log.Debug().Msgf("Log level set to %s", logLevel)
	log.Info().Msgf("Log level set to %s", logLevel)
	log.Warn().Msgf("Log level set to %s", logLevel)
	log.Error().Msgf("Log level set to %s", logLevel)

	fmt.Println()
}

func main() {
	testLogLevel("")
	testLogLevel("trace")
	testLogLevel("debug")
	testLogLevel("info")
	testLogLevel("warn")
	testLogLevel("error")
	testLogLevel("somethingelse")
}
