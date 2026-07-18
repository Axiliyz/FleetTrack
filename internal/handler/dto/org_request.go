package dto

import "fleettrack/internal/model"

// OrgRequest описывает тело HTTP запроса при создании Organization
type OrgRequest struct {
	Name string `json:"name"`
}

// ToDomainModel конвертирует OrgRequest в domain-модель model.Org
func (r *OrgRequest) ToDomainModel() model.Org {
	return model.Org{
		Name: r.Name,
	}
}
