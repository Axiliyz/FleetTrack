package dto

import "fleettrack/internal/model"

// UserResponse — представление пользователя в API-ответах
type UserResponse struct {
	ID             int            `json:"id"`
	OrganizationID int            `json:"organization_id"`
	Name           string         `json:"name"`
	Email          string         `json:"email"`
	Role           model.UserRole `json:"role"`
}

// NewUserResponse собирает UserResponse из доменной модели model.User
func NewUserResponse(u model.User) UserResponse {
	return UserResponse{
		ID:             u.ID,
		OrganizationID: u.OrganizationID,
		Name:           u.Name,
		Email:          u.Email,
		Role:           u.Role,
	}
}
