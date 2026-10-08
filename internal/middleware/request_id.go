// Package middleware для автоматической обработки запросов
package middleware

import (
	"fleettrack/internal/requestid"
	"net/http"

	"github.com/google/uuid"
)

const maxRequestIDLen = 128

// RequestID берёт ID запроса из заголовка X-Request-ID или генерирует новый,
// кладёт его в контекст и возвращает клиенту в том же заголовке.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestid.Header)
		if id == "" || len(id) > maxRequestIDLen {
			id = uuid.NewString()
		}
		w.Header().Set(requestid.Header, id)
		next.ServeHTTP(w, r.WithContext(requestid.NewContext(r.Context(), id)))
	})
}
