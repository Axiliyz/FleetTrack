package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestBodyLimit проверяет, что тело больше предела не дочитывается, а меньше - проходит
func TestBodyLimit(t *testing.T) {
	var decodeErr error
	h := BodyLimit(64)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var v map[string]string
		decodeErr = json.NewDecoder(r.Body).Decode(&v)
	}))

	small := `{"email":"a@b.c"}`
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(small)))
	if decodeErr != nil {
		t.Fatalf("small body: %v", decodeErr)
	}

	big := `{"email":"` + strings.Repeat("a", 1000) + `"}`
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(big)))
	var tooLarge *http.MaxBytesError
	if !errors.As(decodeErr, &tooLarge) {
		t.Fatalf("big body err = %v, want *http.MaxBytesError", decodeErr)
	}
}
