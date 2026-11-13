package dto

import "fleettrack/internal/model"

// CreateUserRequest - тело запроса на создание юзера.
// Организации в теле нет: юзер создаётся в организации админа из токена
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	// DriverID - водитель, к которому привязан аккаунт. Обязателен для роли DRIVER
	// и запрещён для остальных; водитель должен быть из той же организации
	DriverID *int `json:"driver_id,omitempty"`
}

// ToDomainModel переводит CreateUserRequest в доменную модель юзера организации organizationID
func (r *CreateUserRequest) ToDomainModel(organizationID int) model.User {
	return model.User{
		OrganizationID: organizationID,
		DriverID:       r.DriverID,
		Name:           r.Name,
		Email:          r.Email,
		Role:           model.UserRole(r.Role),
	}
}
