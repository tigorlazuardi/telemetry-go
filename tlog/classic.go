package tlog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// ClassicLogger provides log.Print-style compatibility helpers.
//
// It always logs with context.Background().
type ClassicLogger struct {
	logger *slog.Logger
}

// Classic creates a classic compatibility logger.
func Classic() *ClassicLogger { return &ClassicLogger{} }

// Logger sets the logger used by the classic compatibility helpers.
func (c *ClassicLogger) Logger(logger *slog.Logger) *ClassicLogger {
	c.logger = logger
	return c
}

func (c *ClassicLogger) log(level slog.Leveler, msg string) {
	Log(context.Background(), Opt().Logger(c.logger).Level(level).setMessage(msg))
}

func (c *ClassicLogger) logf(level slog.Leveler, format string, args ...any) {
	Log(context.Background(), Opt().Logger(c.logger).Level(level).Message(format, args...))
}

// Print logs with context.Background() at info level.
func (c *ClassicLogger) Print(args ...any) { c.log(slog.LevelInfo, fmt.Sprint(args...)) }

// Printf logs with context.Background() at info level.
func (c *ClassicLogger) Printf(format string, args ...any) { c.logf(slog.LevelInfo, format, args...) }

// Println logs with context.Background() at info level.
func (c *ClassicLogger) Println(args ...any) {
	c.log(slog.LevelInfo, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
}

// Debug logs with context.Background() at debug level.
func (c *ClassicLogger) Debug(args ...any) { c.log(slog.LevelDebug, fmt.Sprint(args...)) }

// Debugf logs with context.Background() at debug level.
func (c *ClassicLogger) Debugf(format string, args ...any) { c.logf(slog.LevelDebug, format, args...) }

// Info logs with context.Background() at info level.
func (c *ClassicLogger) Info(args ...any) { c.log(slog.LevelInfo, fmt.Sprint(args...)) }

// Infof logs with context.Background() at info level.
func (c *ClassicLogger) Infof(format string, args ...any) { c.logf(slog.LevelInfo, format, args...) }

// Warn logs with context.Background() at warn level.
func (c *ClassicLogger) Warn(args ...any) { c.log(slog.LevelWarn, fmt.Sprint(args...)) }

// Warnf logs with context.Background() at warn level.
func (c *ClassicLogger) Warnf(format string, args ...any) { c.logf(slog.LevelWarn, format, args...) }

// Error logs with context.Background() at error level.
func (c *ClassicLogger) Error(args ...any) { c.log(slog.LevelError, fmt.Sprint(args...)) }

// Errorf logs with context.Background() at error level.
func (c *ClassicLogger) Errorf(format string, args ...any) { c.logf(slog.LevelError, format, args...) }

// Print logs with context.Background() at info level.
func Print(args ...any) { Classic().Print(args...) }

// Printf logs with context.Background() at info level.
func Printf(format string, args ...any) { Classic().Printf(format, args...) }

// Println logs with context.Background() at info level.
func Println(args ...any) { Classic().Println(args...) }
