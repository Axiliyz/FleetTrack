package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// UserHandler обрабатывает запросы, связанные с юзерами
type UserHandler struct {
	userService UserService
	logger      logger.Logger
}

// UserService определяет контракт бизнес-логики, необходимой UserHandler
type UserService interface {
	// CreateUser валидирует и сохраняет нового пользователя
	CreateUser(ctx context.Context, u model.User, password string) (model.User, error)
	// GetUserByID возвращает пользователя по его ID
	GetUserByID(ctx context.Context, id int) (model.User, error)
	// GetUserByEmail возвращает пользователя по email
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	// GetUsersList возвращает список пользователей по фильтру
	GetUsersList(ctx context.Context, filter model.UserFilter) ([]model.User, error)
	// DeleteUserByID удаляет пользователя по его ID
	DeleteUserByID(ctx context.Context, id int, organizationID *int) (model.User, error)
}

func NewUserHandler(s UserService, l logger.Logger) *UserHandler {
	return &UserHandler{
		userService: s,
		logger:      l,
	}
}

// HandleCreateUser обрабатывает POST запрос на создание нового пользователя
func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var userData dto.CreateUserRequest
	err := json.NewDecoder(r.Body).Decode(&userData)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	newUser := userData.ToDomainModel()
	newUser.OrganizationID = authCtx.OrganizationID
	savedUser, err := h.userService.CreateUser(r.Context(), newUser, userData.Password)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondSuccess(w, r, "user created", h.logger, dto.NewUserResponse(savedUser))
}

// HandleGetUserByID получает пользователя по ID
func (h *UserHandler) HandleGetUserByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidUserID)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	user, err := h.userService.GetUserByID(r.Context(), id)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	if err := requireOwnOrg(authCtx, user.OrganizationID); err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondSuccess(w, r, "user found", h.logger, dto.NewUserResponse(user))
}

// HandleGetListUsers возвращает список пользователей с фильтрами
func (h *UserHandler) HandleGetListUsers(w http.ResponseWriter, r *http.Request) {
	filter, err := dto.ParseUserFilter(r.URL.Query())
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}
	filter.OrganizationID = scopeOrganizationID(authCtx)

	users, err := h.userService.GetUsersList(r.Context(), filter)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}

	responses := make([]dto.UserResponse, 0, len(users))
	for _, u := range users {
		responses = append(responses, dto.NewUserResponse(u))
	}
	respondSuccess(w, r, "users list", h.logger, responses)
}

// HandleDeleteUserByID обрабатывает DELETE запрос на удаление пользователя по ID
func (h *UserHandler) HandleDeleteUserByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidUserID)
		return
	}

	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}
	orgScope := scopeOrganizationID(authCtx)

	deletedUser, err := h.userService.DeleteUserByID(r.Context(), id, orgScope)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondSuccess(w, r, "user deleted", h.logger, dto.NewUserResponse(deletedUser))
}
