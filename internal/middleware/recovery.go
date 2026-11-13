package middleware

import (
	"fleettrack/internal/httpresp"
	"fleettrack/internal/logger"
	"fmt"
	"net/http"
)

// Recovery перехватывает панику и отправляет 500 ответ
func Recovery(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() { //nolint:contextcheck // контекст запроса передаётся через r
				if rec := recover(); rec != nil {
					log.Log(r.Context(), logger.ErrorLevel, "panic",
						"method", r.Method, "path", r.URL.Path, "panic", fmt.Sprint(rec))
					httpresp.Error(w, r, log, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
