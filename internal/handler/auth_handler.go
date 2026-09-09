package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"net/http"
)

type AuthHandler struct {
	authService AuthService
	logger      logger.Logger
}

type AuthService interface {
	Login(ctx context.Context, email string, password string) (model.AuthResult, error)
	RegisterCompany(ctx context.Context, orgName, adminName, adminEmail, adminPassword string) (model.AuthResult, error)
	Refresh(ctx context.Context, refreshToken string) (model.AuthResult, error)
	Logout(ctx context.Context, refreshToken string) error
}

func NewAuthHandler(s AuthService, l logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: s,
		logger:      l,
	}
}

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

func (h *AuthHandler) HandleRegisterCompany(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request dto.RegisterCompanyRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondError(w, r, h.logger, model.ErrInvalidJSON)
		return
	}

	result, err := h.authService.RegisterCompany(r.Context(), request.OrganizationName, request.AdminName, request.AdminEmail, request.AdminPassword)
	if err != nil {
		respondError(w, r, h.logger, err)
		return
	}
	respondSuccess(w, r, "company registered", h.logger, result)
}

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
