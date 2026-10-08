package logger

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
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

// StdLogger определяет стандартный логгер
type StdLogger struct {
	level Level
}

var _ Logger = (*StdLogger)(nil)

// NewStdLogger создаёт новый стандартный логгер
func NewStdLogger(level Level) *StdLogger {
	return &StdLogger{level: level}
}

// timestamp для форматирование дата-времени
// Возвращает форматированный Now
func timestamp() string {
	return time.Now().Format(time.RFC3339Nano)
}

// Debug уровень для логирования
func (l *StdLogger) Debug(message string) {
	if l.level > DebugLevel {
		return
	}
	color.Cyan("[DEBUG] %s : %s\n", timestamp(), message)
}

// Info уровень для логирования
func (l *StdLogger) Info(message string) {
	if l.level > InfoLevel {
		return
	}
	fmt.Printf("[INFO] %s | %s\n", timestamp(), message)
}

// Warn уровень для логирования
func (l *StdLogger) Warn(message string) {
	if l.level > WarnLevel {
		return
	}
	color.Yellow("[WARN] %s | %s\n", timestamp(), message)
}

// Error уровень для логирования
func (l *StdLogger) Error(message string) {
	if l.level > ErrorLevel {
		return
	}
	color.Red("[ERROR] %s | %s\n", timestamp(), message)
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
