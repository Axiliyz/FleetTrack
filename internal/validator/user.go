package validator

import (
	"fleettrack/internal/model"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 72
)

// IsRoleValid определяет, входит ли роль пользователя в допустимый enum
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

// ValidatePassword проверяет корректность пароля: от 8 символов, не больше 72 байт, хотя бы 1 буква и цифра.
// Минимум — в символах, как его понимает человек. Максимум — в байтах: bcrypt молча отбрасывает
// всё после 72-го байта, и длинный кириллический пароль тихо укоротился бы
func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordLen || len(password) > maxPasswordLen {
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
