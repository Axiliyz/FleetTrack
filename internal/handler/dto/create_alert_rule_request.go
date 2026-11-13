package dto

import "fleettrack/internal/model"

// CreateAlertRuleRequest — тело запроса POST /alert-rules.
// Организации в теле нет: правило всегда создаётся в организации из токена
type CreateAlertRuleRequest struct {
	Type      model.AlertRuleType `json:"type"`
	Name      string              `json:"name"`
	Threshold float64             `json:"threshold"`
	Severity  model.SeverityLevel `json:"severity"`
	Enabled   *bool               `json:"enabled"`
}

// ToDomainModel конвертирует запрос в доменную модель model.AlertRule организации organizationID,
// подставляя Enabled=true по умолчанию
func (r *CreateAlertRuleRequest) ToDomainModel(organizationID int) model.AlertRule {
	enabled := true
	if r.Enabled != nil {
		enabled = *r.Enabled
	}
	return model.AlertRule{
		OrganizationID: organizationID,
		Type:           r.Type,
		Name:           r.Name,
		Threshold:      r.Threshold,
		Severity:       r.Severity,
		Enabled:        enabled,
	}
}
