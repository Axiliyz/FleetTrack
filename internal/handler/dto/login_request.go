package dto

// LoginRequest — тело запроса POST /login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
