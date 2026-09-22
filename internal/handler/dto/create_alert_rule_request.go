package dto

import "fleettrack/internal/model"

// CreateAlertRuleRequest — тело запроса POST /alert-rules
type CreateAlertRuleRequest struct {
	OrganizationID int                 `json:"organization_id"`
	Type           model.AlertRuleType `json:"type"`
	Name           string              `json:"name"`
	Threshold      float64             `json:"threshold"`
	Severity       model.SeverityLevel `json:"severity"`
	Enabled        *bool               `json:"enabled"`
}

// ToDomainModel конвертирует запрос в доменную модель model.AlertRule, подставляя Enabled=true по умолчанию
func (r *CreateAlertRuleRequest) ToDomainModel() model.AlertRule {
	enabled := true
	if r.Enabled != nil {
		enabled = *r.Enabled
	}
	return model.AlertRule{
		OrganizationID: r.OrganizationID,
		Type:           r.Type,
		Name:           r.Name,
		Threshold:      r.Threshold,
		Severity:       r.Severity,
		Enabled:        enabled,
	}
}
