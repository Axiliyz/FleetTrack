package model

import "github.com/golang-jwt/jwt/v5"

// JWTClaims — набор данных, зашиваемых в access-токен
type JWTClaims struct {
	UserID         int      `json:"user_id"`
	OrganizationID int      `json:"organization_id"`
	DriverID       *int     `json:"driver_id"`
	Role           UserRole `json:"role"`

	jwt.RegisteredClaims
}
