package dto

import "fleettrack/internal/model"

// UpdateAlertRequest описывает тело PATCH /alerts/{id}.
// Пока поддерживается только перевод в ACKNOWLEDGED — RESOLVED система ставит сама по телеметрии
type UpdateAlertRequest struct {
	Status model.AlertStatus `json:"status" enums:"ACKNOWLEDGED" example:"ACKNOWLEDGED"`
}

// Validate проверяет, что запрошен поддерживаемый переход статуса
func (u UpdateAlertRequest) Validate() error {
	if u.Status != model.AlertStatusAcknowledged {
		return model.ErrInvalidStatus
	}
	return nil
}
