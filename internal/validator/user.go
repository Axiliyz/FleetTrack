package validator

import (
	"fleettrack/internal/model"
	"net/mail"
	"strings"
	"unicode"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 72
)

// isRoleValid определяет, входит ли роль пользователя в допустимый enum
func IsRoleValid(r model.UserRole) error {
	switch r {
	case model.UserRoleAdmin, model.UserRoleAnalytic, model.UserRoleDispatcher, model.UserRoleDriver:
		return nil
	default:
		return model.ErrInvalidUserRole
	}
}

// ValidateEmail проверяет корректность почты
func ValidateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return model.ErrInvalidEmail
	}
	return nil
}

// ValidatePassword проверяет корректность пароля (длина [8:72] и хотя бы 1 буква и цифра)
func ValidatePassword(password string) error {
	if len(password) < minPasswordLen || len(password) > maxPasswordLen {
		return model.ErrForbiddenPassword
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasDigit || !hasLetter {
		return model.ErrForbiddenPassword
	}
	return nil
}

// CheckPasswordLegit проверяет не дублирует ли пароль почту/имя
func CheckPasswordLegit(password, email, name string) error {
	lower := strings.ToLower(password)
	if lower == strings.ToLower(email) || lower == strings.ToLower(name) {
		return model.ErrForbiddenPassword
	}

	return nil
}
