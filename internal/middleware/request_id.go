// Package middleware для выставления uuid запроса
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type key string

const RequestIDKey key = "request_id"

// RequestID для проброса хэндлера дальше
// Возвращает новый хэндлер
func RequestID(next http.Handler) http.Handler {

	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {

		id := uuid.New().String()

		ctx := context.WithValue(
			r.Context(),
			RequestIDKey,
			id,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}
