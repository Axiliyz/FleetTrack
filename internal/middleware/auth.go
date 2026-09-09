package middleware

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"net/http"
	"strings"
)

type ctxKey string

// authContextKey - ключ контекста, по которому хранится model.AuthContext аутентифицированного пользователя
const authContextKey ctxKey = "auth_context"

// JWTParser описывает то, что AuthMiddleware нужно от JWT-сервиса.
// Объявлен здесь, а не в service, чтобы middleware не зависел от конкретной реализации (DIP) -
// *service.JWTService реализует этот интерфейс неявно.
type JWTParser interface {
	Parse(tokenString string) (*model.JWTClaims, error)
}

// AuthMiddleware проверяет Bearer-токен и кладёт данные пользователя (UserID, Role) в контекст запроса.
// Проверку прав доступа (кому что можно) сюда не добавляем - для этого отдельная RequireRole middleware,
// которая читает model.AuthContext, положенный здесь.
func AuthMiddleware(jwtParser JWTParser, log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				respondUnauthorized(w, r, log, model.ErrMissingToken)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := jwtParser.Parse(tokenString)
			if err != nil {
				respondUnauthorized(w, r, log, err)
				return
			}

			authCtx := model.AuthContext{
				UserID:         claims.UserID,
				OrganizationID: claims.OrganizationID,
				DriverID:       claims.DriverID,
				Role:           claims.Role,
			}
			ctx := context.WithValue(r.Context(), authContextKey, authCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthFromContext достаёт model.AuthContext, положенный AuthMiddleware.
// Используется хендлерами и RequireRole для получения UserID/Role текущего запроса.
func AuthFromContext(ctx context.Context) (model.AuthContext, bool) {
	authCtx, ok := ctx.Value(authContextKey).(model.AuthContext)
	return authCtx, ok
}

// ContextWithAuth кладёт model.AuthContext в контекст напрямую, минуя разбор JWT.
// Нужен тестам хендлеров, которые дёргают http.HandlerFunc напрямую, не пропуская запрос
// через настоящий AuthMiddleware.
func ContextWithAuth(ctx context.Context, authCtx model.AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, authCtx)
}

// respondUnauthorized логирует ошибку аутентификации и отправляет JSON-ответ 401
// в том же формате, что и остальные обработчики (см. handler.writeError)
func respondUnauthorized(w http.ResponseWriter, r *http.Request, log logger.Logger, err error) {
	log.Warn(err.Error())

	id, ok := r.Context().Value(RequestIDKey).(string)
	if !ok {
		id = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	if encErr := json.NewEncoder(w).Encode(dto.ErrorResponse{
		Status:    "error",
		Message:   "unauthorized",
		RequestID: id,
	}); encErr != nil {
		log.Error(encErr.Error())
	}
}
