package middleware

import (
	"fleettrack/internal/logger"
	"fmt"
	"net/http"
)

// Recovery перехватывает панику и отправляет 500 ответ
func Recovery(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			defer func() {
				ctx := r.Context()
				id, ok := ctx.Value(
					RequestIDKey,
				).(string)

				if !ok {
					id = "unknown"
				}

				if rec := recover(); rec != nil {
					logger.Error(fmt.Sprintf("panic:\nrequest_id = %s, \nmethod = %s \nURL = %s, \npanic = %v", id, r.Method, r.URL.Path, rec))

					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
