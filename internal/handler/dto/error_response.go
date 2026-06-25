package dto

type ErrorResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}
