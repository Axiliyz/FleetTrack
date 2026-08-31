package dto

import "fleettrack/internal/model"

// UpdateDriverRequest описывает тело HTTP запроса на частичное обновление водителя
type UpdateDriverRequest struct {
	OrganizationID *int    `json:"organization_id"`
	Name           *string `json:"name"`
}

// ToDomainModel преобразует UpdateDriverRequest в доменную модель
func (u *UpdateDriverRequest) ToDomainModel() model.UpdateDriver {
	return model.UpdateDriver{
		OrganizationID: u.OrganizationID,
		Name:           u.Name,
	}
}
