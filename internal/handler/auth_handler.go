package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"net/http"
)

// AuthHandler обрабатывает HTTP-запросы аутентификации: логин, регистрацию
// организации, ротацию refresh-токена и логаут
type AuthHandler struct {
	authService AuthService
	logger      logger.Logger
}

// AuthService описывает бизнес-логику аутентификации, которую использует AuthHandler
type AuthService interface {
	Login(ctx context.Context, email string, password string) (model.AuthResult, error)
	RegisterOrganization(ctx context.Context, orgName, adminName, adminEmail, adminPassword string) (model.AuthResult, error)
	Refresh(ctx context.Context, refreshToken string) (model.AuthResult, error)
	Logout(ctx context.Context, refreshToken string) error
}

// NewAuthHandler создаёт AuthHandler с переданными сервисом аутентификации и логгером
func NewAuthHandler(s AuthService, l logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: s,
		logger:      l,
	}
}

// HandleLogin обрабатывает POST /login: проверяет учётные данные и возвращает пару токенов
// @Summary Вход в систему
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Учётные данные"
// @Success 200 {object} model.AuthResult
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /login [post]
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request dto.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	result, err := h.authService.Login(r.Context(), request.Email, request.Password)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondSuccess(w, r, "logged in", h.logger, result)
}

// HandleRegisterOrganization обрабатывает POST /register: создаёт организацию и её первичного администратора
// @Summary Регистрация компании
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterOrganizationRequest true "Данные организации"
// @Success 201 {object} model.AuthResult
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /register [post]
func (h *AuthHandler) HandleRegisterOrganization(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request dto.RegisterOrganizationRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	result, err := h.authService.RegisterOrganization(r.Context(), request.OrganizationName, request.AdminName, request.AdminEmail, request.AdminPassword)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondCreated(w, r, "Organization registered", h.logger, result)
}

// HandleRefresh обрабатывает POST /refresh: ротирует пару access/refresh токенов по валидному refresh-токену
// @Summary Обновление токенов доступа и рефреша
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Рефреш токен"
// @Success 200 {object} model.AuthResult
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /refresh [post]
func (h *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request dto.RefreshTokenRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	result, err := h.authService.Refresh(r.Context(), request.RefreshToken)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondSuccess(w, r, "token refreshed", h.logger, result)
}

// HandleLogout обрабатывает POST /logout: отзывает текущую сессию refresh-токена
// @Summary Выход из системы
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Рефреш токен"
// @Success 200
// @Failure 400 {object} dto.ErrorResponse
// @Router /logout [post]
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request dto.RefreshTokenRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	if err := h.authService.Logout(r.Context(), request.RefreshToken); err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondSuccess(w, r, "logged out", h.logger, nil)
}
