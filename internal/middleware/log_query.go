package middleware

import (
	"fleettrack/internal/logger"
	"net/http"
	"time"
)

// unloggedPaths — пути, которые LogQuery не пишет в лог (пробы оркестратора
// вызываются часто и засоряли бы вывод)
var unloggedPaths = map[string]struct{}{
	"/health": {},
	"/readyz": {},
}

// LogQuery - middleware, логирующее каждый обработанный HTTP запрос одной структурной записью:
// метод, путь, статус, время, а также request_id, user_id и org_id.
// Успешные чтения пишутся на уровне Debug - их большинство, и в Info они топили бы всё остальное
func LogQuery(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := unloggedPaths[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}
			// контейнер полей кладём до next: AuthMiddleware ниже по цепочке допишет в него пользователя
			ctx := logger.WithRequestFields(r.Context())
			r = r.WithContext(ctx)

			start := time.Now()
			rw := &ResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}
			next.ServeHTTP(rw, r)

			log.Log(ctx, requestLogLevel(r.Method, rw.statusCode), "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration_ms", float64(time.Since(start).Microseconds())/1000,
			)
		})
	}
}

// requestLogLevel выбирает уровень записи о запросе по методу и статусу ответа
func requestLogLevel(method string, status int) logger.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return logger.ErrorLevel
	case status >= http.StatusBadRequest:
		return logger.WarnLevel
	case method == http.MethodGet || method == http.MethodHead:
		return logger.DebugLevel
	default:
		return logger.InfoLevel
	}
}
