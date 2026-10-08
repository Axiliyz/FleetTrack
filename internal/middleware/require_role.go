package middleware

import (
	"fleettrack/internal/httpresp"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fmt"
	"net/http"
)

// RequireRole возвращает middleware, пропускающий запрос только если роль из контекста аутентификации входит в roles
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
	httpresp.Error(w, r, log, http.StatusForbidden, "forbidden")
}
