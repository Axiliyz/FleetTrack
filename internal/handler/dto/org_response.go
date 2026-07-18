package dto

import "fleettrack/internal/model"

// OrgResponse описывает HTTP ответ при создании организации
type OrgResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// NewOrgResponse - конструктор для HTTP ответа по организации
func NewOrgResponse(o model.Org) OrgResponse {
	return OrgResponse{
		ID:   o.ID,
		Name: o.Name,
	}
}
