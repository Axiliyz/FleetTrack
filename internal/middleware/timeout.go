package middleware

import (
	"context"
	"net/http"
	"time"
)

// TimeoutMiddleware ограничивает время обработки запроса, отменяя контекст
// запроса по истечении timeout.
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
