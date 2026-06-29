// Package logger определяет интерфейс самописного логгера
package logger

// Logger определяет контракт логгера
type Logger interface {
	Debug(message string)
	Info(message string)
	Warn(message string)
	Error(message string)
}
