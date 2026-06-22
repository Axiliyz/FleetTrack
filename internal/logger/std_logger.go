package logger

import (
	"fmt"
	"time"

	"github.com/fatih/color"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type StdLogger struct {
	level Level
}

func NewStdLogger(level Level) *StdLogger {
	return &StdLogger{level: level}
}

func timestamp() string {
	return time.Now().Format(time.RFC3339Nano)
}

func (l *StdLogger) Debug(message string) {
	if l.level > DebugLevel {
		return
	}
	color.Cyan("[DEBUG] %s : %s\n", timestamp(), message)
}

func (l *StdLogger) Info(message string) {
	if l.level > InfoLevel {
		return
	}
	fmt.Printf("[INFO] %s | %s\n", timestamp(), message)
}

func (l *StdLogger) Warn(message string) {
	if l.level > WarnLevel {
		return
	}
	color.Yellow("[WARN] %s | %s\n", timestamp(), message)
}

func (l *StdLogger) Error(message string) {
	if l.level > ErrorLevel {
		return
	}
	color.Red("[ERROR] %s | %s\n", timestamp(), message)
}
