// Package logadapter bridges a zerolog.Logger into the ad hoc Write,
// Printf/Println, and Errorf interfaces that third-party libraries expect
// from a logger (io.Writer, go-smtp's Logger, go-socks5's Logger,
// tcpproxy's Logger, and the standard library's *log.Logger).
//
// Construct an Adapter from a logger that already carries any fixed
// context (component, module, tunnel ID, etc.) via zerolog's With(); the
// Adapter itself adds no context beyond the level at which it logs.
package logadapter

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

// Adapter adapts a zerolog.Logger to the logging interfaces expected by
// third-party libraries. A single Adapter value satisfies io.Writer, the
// Printf/Println-style Logger interface used by go-smtp and tcpproxy, and
// the Errorf-style Logger interface used by go-socks5 — pick whichever
// method(s) the target interface requires.
//
// All methods log a single event at Level, with any trailing newline
// trimmed since callers typically pass pre-formatted lines.
type Adapter struct {
	Logger zerolog.Logger
	Level  zerolog.Level
}

// New returns an Adapter that logs to logger at level.
func New(logger zerolog.Logger, level zerolog.Level) Adapter {
	return Adapter{Logger: logger, Level: level}
}

func (a Adapter) log(msg string) {
	a.Logger.WithLevel(a.Level).Msg(strings.TrimRight(msg, "\n"))
}

// Write implements io.Writer.
func (a Adapter) Write(p []byte) (int, error) {
	a.log(string(p))
	return len(p), nil
}

// Printf implements the Printf half of the Logger interfaces used by
// go-smtp and tcpproxy.
func (a Adapter) Printf(format string, v ...any) {
	a.log(fmt.Sprintf(format, v...))
}

// Println implements the Println half of go-smtp's Logger interface.
func (a Adapter) Println(v ...any) {
	a.log(fmt.Sprintln(v...))
}

// Errorf implements go-socks5's Logger interface.
func (a Adapter) Errorf(format string, v ...any) {
	a.log(fmt.Sprintf(format, v...))
}
