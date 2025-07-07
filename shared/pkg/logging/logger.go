package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
	With(any ...any) Logger
	WithGroup(name string) Logger
}

type slogLogger struct {
	logger *slog.Logger
}

func NewLogger(config *Config) Logger {
	var level slog.Level
	switch config.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: config.AddSource,
	}

	var handler slog.Handler
	switch config.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "console", "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)

	return &slogLogger{
		logger: logger,
	}
}

func NewNoop() Logger {
	handler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError + 1,
	})
	return &slogLogger{
		logger: slog.New(handler),
	}
}

func NewDefault() Logger {
	return NewLogger(&Config{
		Level:     "info",
		Format:    "json",
		AddSource: false,
	})
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *slogLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, args...)
}

func (l *slogLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, args...)
}

func (l *slogLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, args...)
}

func (l *slogLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, args...)
}

func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{
		logger: l.logger.With(args...),
	}
}

func (l *slogLogger) WithGroup(name string) Logger {
	return &slogLogger{
		logger: l.logger.WithGroup(name),
	}
}

type Config struct {
	Level     string `yaml:"level" json:"level" default:"info"`
	Format    string `yaml:"format" json:"format" default:"json"`
	AddSource bool   `yaml:"add_source" json:"add_source" default:"false"`
}

func (c *Config) Validate() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[c.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn or error)", c.Level)
	}

	validFormats := map[string]bool{
		"json":    true,
		"console": true,
		"text":    true,
	}

	if !validFormats[c.Format] {
		return fmt.Errorf("invalid log format: %s (must be json, console or text", c.Format)
	}

	return nil
}

func DefaultConfig() *Config {
	return &Config{
		Level:     "info",
		Format:    "json",
		AddSource: false,
	}
}

func DevelopmentConfig() *Config {
	return &Config{
		Level:     "debug",
		Format:    "console",
		AddSource: true,
	}
}

func ProductionConfig() *Config {
	return &Config{
		Level:     "info",
		Format:    "json",
		AddSource: false,
	}
}
