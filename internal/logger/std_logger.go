package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Level определяет уровень логгера
type Level int

// Enum с уровнями логгера
const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// slogLevel переводит уровень логгера в уровень slog
func (l Level) slogLevel() slog.Level {
	switch l {
	case DebugLevel:
		return slog.LevelDebug
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Format - формат вывода логов
type Format string

const (
	// FormatJSON - одна JSON-запись на строку, для сборщиков логов (Loki, ELK)
	FormatJSON Format = "json"
	// FormatText - key=value, удобнее читать глазами при локальной разработке
	FormatText Format = "text"
)

// StdLogger определяет стандартный логгер: обёртка над *slog.Logger
type StdLogger struct {
	l *slog.Logger
}

var _ Logger = (*StdLogger)(nil)

// NewStdLogger создаёт логгер, пишущий JSON в stdout
func NewStdLogger(level Level) *StdLogger {
	return New(os.Stdout, level, FormatJSON)
}

// New создаёт логгер, пишущий в w в формате format
func New(w io.Writer, level Level, format Format) *StdLogger {
	opts := &slog.HandlerOptions{Level: level.slogLevel()}
	var h slog.Handler
	if format == FormatText {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return &StdLogger{l: slog.New(contextHandler{h})}
}

// Debug уровень для логирования
func (l *StdLogger) Debug(message string) {
	l.l.Debug(message)
}

// Info уровень для логирования
func (l *StdLogger) Info(message string) {
	l.l.Info(message)
}

// Warn уровень для логирования
func (l *StdLogger) Warn(message string) {
	l.l.Warn(message)
}

// Error уровень для логирования
func (l *StdLogger) Error(message string) {
	l.l.Error(message)
}

// Log пишет запись с полями args и полями запроса из ctx
func (l *StdLogger) Log(ctx context.Context, level Level, message string, args ...any) {
	l.l.Log(ctx, level.slogLevel(), message, args...)
}

// ParseLevel разбирает уровень логирования из строки: debug, info, warn, error.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return DebugLevel, nil
	case "info", "":
		return InfoLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	}
	return InfoLevel, fmt.Errorf("unknown log level %q", s)
}

// ParseFormat разбирает формат логов из строки: json (по умолчанию) или text
func ParseFormat(s string) (Format, error) {
	switch Format(strings.ToLower(strings.TrimSpace(s))) {
	case FormatJSON, "":
		return FormatJSON, nil
	case FormatText:
		return FormatText, nil
	}
	return FormatJSON, fmt.Errorf("unknown log format %q", s)
}
