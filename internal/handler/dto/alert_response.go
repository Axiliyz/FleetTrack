package dto

import (
	"fleettrack/internal/model"
	"time"
)

// AlertResponse — представление алерта в API-ответах
type AlertResponse struct {
	ID             int                 `json:"id"`
	OrganizationID int                 `json:"organization_id"`
	VehicleID      int                 `json:"vehicle_id"`
	RuleID         int                 `json:"rule_id"`
	Type           model.AlertRuleType `json:"type"`
	Message        string              `json:"message"`
	Severity       model.SeverityLevel `json:"severity"`
	Value          *float64            `json:"value"`
	Status         model.AlertStatus   `json:"status"`
	CreatedAt      time.Time           `json:"created_at"`
	AcknowledgedAt *time.Time          `json:"acknowledged_at"`
	AcknowledgedBy *int                `json:"acknowledged_by"`
	ResolvedAt     *time.Time          `json:"resolved_at"`
	ResolvedBy     *int                `json:"resolved_by"`
}

// NewAlertResponse конвертирует model.Alert в AlertResponse
func NewAlertResponse(a model.Alert) AlertResponse {
	return AlertResponse{
		ID:             a.ID,
		OrganizationID: a.OrganizationID,
		VehicleID:      a.VehicleID,
		RuleID:         a.RuleID,
		Type:           a.Type,
		Message:        a.Message,
		Severity:       a.Severity,
		Value:          a.Value,
		Status:         a.Status,
		CreatedAt:      a.CreatedAt,
		AcknowledgedAt: a.AcknowledgedAt,
		AcknowledgedBy: a.AcknowledgedBy,
		ResolvedAt:     a.ResolvedAt,
		ResolvedBy:     a.ResolvedBy,
	}
}

// NewAlertListResponse конвертирует срез алертов в ответ; пустой список отдаётся как [], а не null
func NewAlertListResponse(alerts []model.Alert) []AlertResponse {
	res := make([]AlertResponse, 0, len(alerts))
	for _, a := range alerts {
		res = append(res, NewAlertResponse(a))
	}
	return res
}
