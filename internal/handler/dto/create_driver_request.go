package dto

import "fleettrack/internal/model"

// CreateDriverRequest описывает тело HTTP запроса на создание водителя.
// Организации в теле нет: водитель создаётся в организации из токена
type CreateDriverRequest struct {
	Name string `json:"name"`
}

// ToDomainModel конвертирует CreateDriverRequest в domain-модель model.Driver организации organizationID
func (r *CreateDriverRequest) ToDomainModel(organizationID int) model.Driver {
	return model.Driver{
		OrganizationID: organizationID,
		Name:           r.Name,
	}
}
