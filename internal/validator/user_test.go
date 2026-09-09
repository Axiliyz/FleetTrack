package validator

import (
	"fleettrack/internal/model"
	"strings"
	"testing"
)

func TestIsRoleValid(t *testing.T) {
	tests := []struct {
		name    string
		role    model.UserRole
		wantErr error
	}{
		{name: "admin", role: model.UserRoleAdmin, wantErr: nil},
		{name: "dispatcher", role: model.UserRoleDispatcher, wantErr: nil},
		{name: "driver", role: model.UserRoleDriver, wantErr: nil},
		{name: "analytic", role: model.UserRoleAnalytic, wantErr: nil},
		{name: "empty", role: "", wantErr: model.ErrInvalidUserRole},
		{name: "unknown", role: "BOSS", wantErr: model.ErrInvalidUserRole},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsRoleValid(tt.role)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{name: "valid", email: "ivan@example.com", wantErr: nil},
		{name: "empty", email: "", wantErr: model.ErrInvalidEmail},
		{name: "no at sign", email: "ivan.example.com", wantErr: model.ErrInvalidEmail},
		{name: "no domain", email: "ivan@", wantErr: model.ErrInvalidEmail},
		{name: "display name not allowed", email: "Ivan <ivan@example.com>", wantErr: model.ErrInvalidEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "valid", password: "Passw0rd", wantErr: nil},
		{name: "too short", password: "p1", wantErr: model.ErrForbiddenPassword},
		{name: "too long", password: strings.Repeat("a1", 40), wantErr: model.ErrForbiddenPassword},
		{name: "at max length", password: "a1" + strings.Repeat("a", 70), wantErr: nil},
		{name: "no digit", password: "passwordonly", wantErr: model.ErrForbiddenPassword},
		{name: "no letter", password: "12345678", wantErr: model.ErrForbiddenPassword},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckPasswordLegit(t *testing.T) {
	tests := []struct {
		name     string
		password string
		email    string
		userName string
		wantErr  error
	}{
		{name: "unrelated password", password: "Passw0rd", email: "ivan@example.com", userName: "Ivan Petrov", wantErr: nil},
		{name: "same as email", password: "ivan@example.com", email: "ivan@example.com", userName: "Ivan Petrov", wantErr: model.ErrForbiddenPassword},
		{name: "same as email different case", password: "IVAN@EXAMPLE.COM", email: "ivan@example.com", userName: "Ivan Petrov", wantErr: model.ErrForbiddenPassword},
		{name: "same as name", password: "Ivan Petrov", email: "ivan@example.com", userName: "Ivan Petrov", wantErr: model.ErrForbiddenPassword},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPasswordLegit(tt.password, tt.email, tt.userName)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}
