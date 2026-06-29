// Package DTO для транспортировки данных
package dto

// ErrorResponse определяет структуру JSON ответа при ошибке
type ErrorResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}
