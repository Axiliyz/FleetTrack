// Package DTO для транспортировки данных
package dto

// APIResponse определяет структуру JSON ответа
type APIResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Data      any    `json:"data"`
}
