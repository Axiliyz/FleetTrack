package logger

import (
	"context"
	"fleettrack/internal/requestid"
	"log/slog"
	"sync"
)

type fieldsKey struct{}

// requestFields - поля запроса, которые middleware дописывают по ходу обработки.
// Контекст неизменяем, а user_id и org_id становятся известны глубже по цепочке
// (в AuthMiddleware), чем пишется строка лога о запросе (в LogQuery). Поэтому LogQuery
// кладёт в контекст изменяемый контейнер, а AuthMiddleware его дополняет
type requestFields struct {
	mu    sync.Mutex
	attrs []any
}

// WithRequestFields кладёт в контекст пустой контейнер полей запроса
func WithRequestFields(ctx context.Context) context.Context {
	return context.WithValue(ctx, fieldsKey{}, &requestFields{})
}

// AddRequestFields дописывает поля (пары ключ-значение) ко всем записям, которые
// дальше будут сделаны с этим контекстом. Без контейнера в ctx ничего не делает
func AddRequestFields(ctx context.Context, args ...any) {
	f, ok := ctx.Value(fieldsKey{}).(*requestFields)
	if !ok {
		return
	}
	f.mu.Lock()
	f.attrs = append(f.attrs, args...)
	f.mu.Unlock()
}

// contextHandler дописывает к записи request_id и поля запроса из контекста
type contextHandler struct {
	slog.Handler
}

// Handle добавляет поля запроса и передаёт запись дальше
func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := requestid.FromContext(ctx); id != requestid.Unknown {
		r.AddAttrs(slog.String("request_id", id))
	}
	if f, ok := ctx.Value(fieldsKey{}).(*requestFields); ok {
		f.mu.Lock()
		r.Add(f.attrs...)
		f.mu.Unlock()
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs сохраняет обёртку при logger.With
func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

// WithGroup сохраняет обёртку при logger.WithGroup
func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}
