// Package logger определяет интерфейс логгера приложения и его реализацию поверх log/slog
package logger

import "context"

// Logger определяет контракт логгера
type Logger interface {
	Debug(message string)
	Info(message string)
	Warn(message string)
	Error(message string)
	// Log пишет запись с полями args (пары ключ-значение, как в slog). Из ctx к записи
	// добавляются поля запроса: request_id и то, что положили через AddRequestFields
	Log(ctx context.Context, level Level, message string, args ...any)
}
