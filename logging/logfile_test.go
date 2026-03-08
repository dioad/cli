package logging_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli/logging"
)

// TestConfigureLogFileOutput verifies the behaviour of ConfigureLogFileOutput
// after it was changed to return (io.Writer, error) instead of io.Writer.
func TestConfigureLogFileOutput(t *testing.T) {
	t.Run("absolute path returns non-nil writer without error", func(t *testing.T) {
		// Arrange — t.TempDir() is isolated per sub-test.
		dir := t.TempDir()
		cfg := logging.Config{
			File:       filepath.Join(dir, "app.log"),
			MaxSize:    10,
			MaxBackups: 2,
			MaxAge:     7,
		}

		// Act
		w, err := logging.ConfigureLogFileOutput(cfg)

		// Assert
		require.NoError(t, err, "a valid absolute path should not return an error")
		assert.NotNil(t, w, "a valid path should return a non-nil io.Writer")
	})

	t.Run("tilde path expands without error", func(t *testing.T) {
		// Arrange — lumberjack opens the file lazily so no file is written.
		cfg := logging.Config{File: "~/testcli-logfile-test.log"}

		// Act
		w, err := logging.ConfigureLogFileOutput(cfg)

		// Assert
		require.NoError(t, err, "a ~/... path should expand without error")
		assert.NotNil(t, w, "tilde expansion should yield a non-nil io.Writer")
	})

	t.Run("unexpandable tilde-user path returns error", func(t *testing.T) {
		// Arrange — ~nonexistentuser_9999 is virtually guaranteed not to exist.
		cfg := logging.Config{File: "~nonexistentuser_9999/app.log"}

		// Act
		w, err := logging.ConfigureLogFileOutput(cfg)

		// Assert
		require.Error(t, err,
			"an unexpandable ~user path should return an error")
		assert.Nil(t, w,
			"an unexpandable path should return a nil io.Writer")
		assert.ErrorContains(t, err, "nonexistentuser_9999",
			"error message should include the problematic path fragment")
	})
}
