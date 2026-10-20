package middleware

import "net/http"

// ResponseWriter оборачивает http.ResponseWriter и запоминает записанный статус-код
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader записывает статус-код ответа и запоминает его для последующего логирования
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write записывает тело ответа, по умолчанию фиксируя статус 200, если WriteHeader ещё не вызывался
func (rw *ResponseWriter) Write(data []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}

	return rw.ResponseWriter.Write(data)
}

// Unwrap отдаёт исходный ResponseWriter. Через него http.ResponseController находит Flush
// и дедлайны соединения — без этого SSE за этой обёрткой не смог бы отправлять события сразу
func (rw *ResponseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
