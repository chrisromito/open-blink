package logger

import (
	"github.com/rs/zerolog"
	"os"
)

var log zerolog.Logger = NewLogger()

// NewLogger creates and returns a new zerolog.Logger instance that writes to stdout.
func NewLogger() zerolog.Logger {
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}

// SetLevel sets the logger's level.
func SetLevel(level zerolog.Level) {
	log = log.Level(level)
}

// GetLogger returns a pointer to the logger.
func GetLogger() *zerolog.Logger {
	return &log
}

// Debug logs a message at debug level.
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info logs a message at info level.
func Info() *zerolog.Event {
	return log.Info()
}

// Warn logs a message at warn level.
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error logs a message at error level.
func Error() *zerolog.Event {
	return log.Error()
}

func Fatal() *zerolog.Event { return log.Fatal() }
