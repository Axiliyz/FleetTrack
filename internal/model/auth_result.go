package model

// AuthResult — пара токенов, выдаваемая при логине, регистрации или refresh
type AuthResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
