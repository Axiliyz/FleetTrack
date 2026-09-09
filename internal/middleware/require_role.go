package middleware

import (
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fmt"
	"net/http"
)

func RequireRole(l logger.Logger, roles ...model.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx, ok := AuthFromContext(r.Context())
			if !ok {
				respondUnauthorized(w, r, l, model.ErrMissingToken)
				return
			}
			allowed := false
			for _, role := range roles {
				if authCtx.Role == role {
					allowed = true
					break
				}
			}
			if !allowed {
				respondForbidden(w, r, l, fmt.Errorf("%w: role %s", model.ErrForbidden, authCtx.Role))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func respondForbidden(w http.ResponseWriter, r *http.Request, log logger.Logger, err error) {
	log.Warn(err.Error())

	id, ok := r.Context().Value(RequestIDKey).(string)
	if !ok {
		id = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)

	if encErr := json.NewEncoder(w).Encode(dto.ErrorResponse{
		Status:    "error",
		Message:   "forbidden",
		RequestID: id,
	}); encErr != nil {
		log.Error(encErr.Error())
	}
}
