// Package requestid хранит идентификатор HTTP-запроса в контексте.
package requestid

import "context"

type ctxKey struct{}

// Header - HTTP-заголовок, через который ID запроса принимается и возвращается клиенту.
const Header = "X-Request-ID"

// Unknown возвращается, если в контексте нет ID запроса.
const Unknown = "unknown"

// NewContext кладёт ID запроса в контекст.
func NewContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext возвращает ID запроса из контекста или Unknown.
func FromContext(ctx context.Context) string {
	if id, ok := ctx.Value(ctxKey{}).(string); ok && id != "" {
		return id
	}
	return Unknown
}
