package dto

import "fleettrack/internal/model"

// CreateUserRequest - тело запроса на создание юзера
type CreateUserRequest struct {
	OrganizationID int    `json:"organization_id"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	Role           string `json:"role"`
}

// ToDomainModel переводит CreateUserRequest в доменную модель юзера
func (r *CreateUserRequest) ToDomainModel() model.User {
	return model.User{
		OrganizationID: r.OrganizationID,
		Name:           r.Name,
		Email:          r.Email,
		Role:           model.UserRole(r.Role),
	}
}
