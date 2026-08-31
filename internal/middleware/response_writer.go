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

func (rw *ResponseWriter) Write(data []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}

	return rw.ResponseWriter.Write(data)
}
