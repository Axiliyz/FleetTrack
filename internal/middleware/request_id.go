package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type key string

const RequestIDKey key = "request_id"

func RequestID(next http.Handler) http.Handler {

	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {

		id := uuid.New().String()

		ctx := context.WithValue(
			r.Context(),
			RequestIDKey,
			id,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}
