package middleware

import (
	"fleettrack/internal/logger"
	"fleettrack/internal/requestid"
	"fmt"
	"net/http"
	"time"
)

// unloggedPaths — пути, которые LogQuery не пишет в лог (пробы оркестратора
// вызываются часто и засоряли бы вывод)
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

			rw := &ResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			logger.Info(fmt.Sprintf("request_id=%s method=%s path=%s status=%d duration=%s",
				requestid.FromContext(r.Context()), r.Method, r.URL.Path, rw.statusCode, duration))
		})
	}

}
