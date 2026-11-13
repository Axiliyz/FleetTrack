package middleware

import (
	"fleettrack/internal/metrics"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// unmatchedRoute - метка пути для запросов, не совпавших ни с одним маршрутом.
// Сырой URL в метку не попадает, чтобы не раздувать кардинальность.
const unmatchedRoute = "unmatched"

// streamRoute - долгоживущий SSE-поток. Его «длительность» - это время, пока открыта вкладка,
// а не время ответа, поэтому в гистограмму латентности он не пишется: иначе p99 всего API
// превратился бы в часы. Сам запрос по-прежнему считается в счётчике.
const streamRoute = "/stream"

// Metrics считает каждый HTTP-запрос по методу, шаблону маршрута и статусу ответа,
// а время обработки пишет в гистограмму.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := ResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		start := time.Now()
		next.ServeHTTP(&wrapped, r)
		duration := time.Since(start)

		path := unmatchedRoute
		if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
			path = rctx.RoutePattern()
		}
		metrics.RecordHTTPRequest(r.Method, path, wrapped.statusCode)
		if path != streamRoute {
			metrics.RecordHTTPDuration(r.Method, path, duration.Seconds())
		}
	})
}
