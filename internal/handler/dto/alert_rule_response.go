package dto

import (
	"fleettrack/internal/model"
	"time"
)

type AlertRuleResponse struct {
	ID             int                 `json:"id"`
	OrganizationID int                 `json:"organization_id"`
	Type           model.AlertRuleType `json:"type"`
	Name           string              `json:"name"`
	Threshold      float64             `json:"threshold"`
	Severity       model.SeverityLevel `json:"severity"`
	Enabled        bool                `json:"enabled"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      *time.Time          `json:"updated_at,omitempty"`
}

// NewAlertRuleResponse конвертирует model.AlertRule в AlertRuleResponse
func NewAlertRuleResponse(r model.AlertRule) AlertRuleResponse {
	return AlertRuleResponse{
		ID:             r.ID,
		OrganizationID: r.OrganizationID,
		Type:           r.Type,
		Name:           r.Name,
		Threshold:      r.Threshold,
		Severity:       r.Severity,
		Enabled:        r.Enabled,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

// NewAlertRuleListResponse конвертирует срез правил в ответ
func NewAlertRuleListResponse(rules []model.AlertRule) []AlertRuleResponse {
	res := make([]AlertRuleResponse, 0, len(rules))
	for _, r := range rules {
		res = append(res, NewAlertRuleResponse(r))
	}
	return res
}
