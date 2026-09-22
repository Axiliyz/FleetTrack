package service

import (
	"fleettrack/internal/model"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService подписывает и разбирает access-токены по алгоритму HS256
type JWTService struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTService создаёт JWTService с ключом подписи secret и временем жизни токена ttl
func NewJWTService(secret string, ttl time.Duration) *JWTService {
	return &JWTService{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// Generate выпускает подписанный access-токен для пользователя user
func (s *JWTService) Generate(user model.User) (string, error) {
	claims := model.JWTClaims{
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		DriverID:       user.DriverID,
		Role:           user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

// Parse проверяет подпись access-токена и возвращает его claims
func (s *JWTService) Parse(tokenString string) (*model.JWTClaims, error) {
	claims := &model.JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, model.ErrInvalidSigningMethod
		}

		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, model.ErrInvalidToken
	}

	return claims, nil
}
