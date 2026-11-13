package middleware

import "net/http"

// DefaultMaxBodyBytes - предел тела запроса. Самые большие тела в API - правило алерта и
// пользователь, это сотни байт; мегабайт с запасом, но не даёт забить память гигабайтным JSON
// на /login и /register, которые доступны без авторизации
const DefaultMaxBodyBytes int64 = 1 << 20

// BodyLimit ограничивает размер тела запроса. Чтение сверх limit возвращает ошибку,
// json.Decoder отдаёт её хендлеру, и тот отвечает 400 invalid json
func BodyLimit(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}
