package service

import (
	"errors"
	"fleettrack/internal/model"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTService_GenerateAndParse(t *testing.T) {
	svc := NewJWTService("test-secret", time.Minute)
	user := model.User{ID: 42, Role: model.UserRoleDispatcher}

	token, err := svc.Generate(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := svc.Parse(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("got UserID %d, want %d", claims.UserID, user.ID)
	}
	if claims.Role != user.Role {
		t.Errorf("got Role %q, want %q", claims.Role, user.Role)
	}
}

func TestJWTService_Parse_Garbage(t *testing.T) {
	svc := NewJWTService("test-secret", time.Minute)

	_, err := svc.Parse("not-a-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestJWTService_Parse_WrongSecret(t *testing.T) {
	genSvc := NewJWTService("secret-a", time.Minute)
	parseSvc := NewJWTService("secret-b", time.Minute)

	token, err := genSvc.Generate(model.User{ID: 1, Role: model.UserRoleDriver})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = parseSvc.Parse(token)
	if err == nil {
		t.Fatal("expected error for token signed with different secret, got nil")
	}
}

func TestJWTService_Parse_Expired(t *testing.T) {
	svc := NewJWTService("test-secret", -time.Minute)

	token, err := svc.Generate(model.User{ID: 1, Role: model.UserRoleDriver})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.Parse(token)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestJWTService_Parse_WrongSigningMethod(t *testing.T) {
	secret := "test-secret"
	svc := NewJWTService(secret, time.Minute)

	claims := model.JWTClaims{
		UserID: 1,
		Role:   model.UserRoleDriver,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("unexpected error signing token: %v", err)
	}

	_, err = svc.Parse(token)
	if err == nil {
		t.Fatal("expected error for unexpected signing method, got nil")
	}
	if !errors.Is(err, model.ErrInvalidSigningMethod) {
		t.Errorf("got %v, want error wrapping %v", err, model.ErrInvalidSigningMethod)
	}
}
