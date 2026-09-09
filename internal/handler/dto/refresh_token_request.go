package dto

// RefreshTokenRequest описывает тело запроса, где нужен refresh-токен -
// используется и для /refresh, и для /logout, форма одинаковая
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}
