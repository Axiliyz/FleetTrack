package dto

import "fleettrack/internal/model"

// CreateDriverRequest описывает тело HTTP запроса на создание водителя
type CreateDriverRequest struct {
	OrganizationID int    `json:"organization_id"`
	Name           string `json:"name"`
}

// ToDomainModel конвертирует CreateDriverRequest в domain-модель model.Driver
func (r *CreateDriverRequest) ToDomainModel() model.Driver {
	return model.Driver{
		OrganizationID: r.OrganizationID,
		Name:           r.Name,
	}
}
