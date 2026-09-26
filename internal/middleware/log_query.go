package middleware

import (
	"fleettrack/internal/logger"
	"fmt"
	"net/http"
	"time"
)

var unloggedPaths = map[string]struct{}{
	"/health": {},
	"/readyz": {},
}

// LogQuery - middleware, логирующее каждый обработанный HTTP запрос
func LogQuery(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := unloggedPaths[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			id, ok := r.Context().Value(RequestIDKey).(string)
			if !ok {
				id = "unknown"
			}
			rw := &ResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			logger.Info(fmt.Sprintf("request_id = %s\nmethod = %s\npath = %s\nstatus=%d\nduration=%s", id, r.Method, r.URL.Path, rw.statusCode, duration))
		})
	}

}
