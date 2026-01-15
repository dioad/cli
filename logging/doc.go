// Package logging provides structured logging configuration and management.
//
// It wraps zerolog to provide:
//
// - Configurable log levels (trace, debug, info, warn, error, fatal, panic)
// - File output with automatic rotation via lumberjack
// - Console output with pretty formatting when attached to a terminal
// - Configuration via Config structs and functional options
//
// # Basic Usage
//
// Configure logging from a config struct:
//
//	cfg := logging.Config{
//		Level:      "debug",
//		File:       "~/.app/logs/app.log",
//		MaxSize:    100,      // MB
//		MaxBackups: 3,        // number of backups
//		MaxAge:     7,        // days
//		Compress:   true,
//	}
//	logging.ConfigureCmdLogger(cfg)
//
// Then use the global zerolog logger:
//
//	import "github.com/rs/zerolog/log"
//
//	log.Info().Msg("application started")
//	log.Debug().Str("user", "alice").Int("count", 42).Msg("event occurred")
//	log.Error().Err(err).Msg("operation failed")
//
// # Log Levels
//
// Supported log levels in order of severity:
//   - trace: most verbose, detailed tracing information
//   - debug: debug information for developers
//   - info: general informational messages
//   - warn: warning messages for potentially problematic situations
//   - error: error messages for failure conditions
//   - fatal: fatal errors that cause application shutdown
//   - panic: panic-level messages
//
// # Configuration
//
// Config struct fields:
//
//	type Config struct {
//		Level      string // Log level: trace, debug, info, warn, error, fatal, panic
//		File       string // File path for log output (empty = stdout only)
//		MaxSize    int    // Max size of log file in MB before rotation
//		MaxAge     int    // Max age of log file in days before deletion
//		MaxBackups int    // Max number of old log files to keep
//		LocalTime  bool   // Use local time in rotated filename timestamps
//		Compress   bool   // Compress old log files
//		Mode       string // Not currently used, reserved for future use
//	}
//
// # File Rotation
//
// When File is configured, logs are written to a file with automatic rotation based on:
//   - MaxSize: File is rotated when it exceeds this size
//   - MaxAge: Rotated files are deleted after this many days
//   - MaxBackups: Only this many old files are retained
//   - Compress: Old rotated files are gzip-compressed if true
//
// Example configuration for production:
//
//	cfg := logging.Config{
//		Level:      "info",
//		File:       "/var/log/myapp/app.log",
//		MaxSize:    500,     // Rotate every 500MB
//		MaxBackups: 10,      // Keep 10 old files
//		MaxAge:     30,      // Delete files older than 30 days
//		Compress:   true,    // Compress rotated files
//	}
//	logging.ConfigureCmdLogger(cfg)
//
// # Output Behavior
//
// The library intelligently selects output format:
//   - Console (TTY): Pretty-printed JSON with timestamps
//   - File/Pipe: Compact JSON for easy parsing
//
// # Functional Options
//
// Configure default log levels with options:
//
//	logging.ConfigureCmdLogger(
//		cfg,
//		logging.WithDefaultLogLevel(zerolog.DebugLevel),
//	)
//
// # Path Expansion
//
// File paths support home directory expansion:
//
//	cfg.File = "~/logs/app.log"  // Automatically expands to user's home dir
//
// # See Also
//
// Package rs/zerolog: structured JSON logging for Go - https://github.com/rs/zerolog
// Package lumberjack: log file rotation and management - https://gopkg.in/natefinch/lumberjack.v2
package logging
