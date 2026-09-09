package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/validator"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// UserService содержит бизнес-логику пользователей
type UserService struct {
	repository repository.UserRepository
	logger     logger.Logger
}

// NewUserService - конструктор сервиса юзеров
func NewUserService(r repository.UserRepository, l logger.Logger) *UserService {
	return &UserService{
		repository: r,
		logger:     l,
	}
}

// GetUserByEmail находит юзера по почте
// Возвращает юзера или ошибку
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	if len(email) < 1 {
		return model.User{}, model.ErrInvalidEmail
	}
	user, err := s.repository.GetByEmail(ctx, email)
	if err != nil {
		return model.User{}, err
	}
	message := fmt.Sprintf("user with email %s found:", email)
	s.logger.Info(message)
	return user, nil
}

// GetUserByID ищет пользователя по ID
// Возвращает объект юзера или ошибку
func (s *UserService) GetUserByID(ctx context.Context, id int) (model.User, error) {
	if id <= 0 {
		return model.User{}, model.ErrInvalidUserID
	}
	user, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return model.User{}, err
	}
	message := fmt.Sprintf("User with id %d:", id)
	s.logger.Info(message)
	return user, nil
}

// validateUser проверяет все поля юзера(кроме пароля, т.к. он зашифрован уже)
func validateUser(u model.User) error {
	if u.OrganizationID < 1 {
		return model.ErrInvalidOrganizationID
	}
	if strings.TrimSpace(u.Name) == "" || len(u.Name) > 35 {
		return model.ErrInvalidName
	}
	if validator.IsRoleValid(u.Role) != nil {
		return model.ErrInvalidUserRole
	}
	if validator.ValidateEmail(u.Email) != nil {
		return model.ErrInvalidEmail
	}
	return nil
}

// CreateUser создаёт пользователя
// Возвращает ошибку при неудаче
func (s *UserService) CreateUser(ctx context.Context, u model.User, password string) (model.User, error) {
	if err := validateUser(u); err != nil {
		return model.User{}, err
	}
	if err := validator.ValidatePassword(password); err != nil {
		return model.User{}, err
	}
	if err := validator.CheckPasswordLegit(password, u.Email, u.Name); err != nil {
		return model.User{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}
	u.PasswordHash = string(hash)
	if err := s.repository.Create(ctx, &u); err != nil {
		return model.User{}, err
	}
	message := fmt.Sprintf("created user with id: %d", u.ID)
	s.logger.Info(message)
	return u, nil
}

// DeleteUserByID удаляет пользователя по его ID
// Возвращает удалённого юзера или ошибку
func (s *UserService) DeleteUserByID(ctx context.Context, id int, organizationID *int) (model.User, error) {
	if id <= 0 {
		return model.User{}, model.ErrInvalidUserID
	}

	user, err := s.repository.DeleteByID(ctx, id, organizationID)
	if err != nil {
		return model.User{}, err
	}
	message := fmt.Sprintf("user with id %d deleted", id)
	s.logger.Info(message)
	return user, nil
}

// GetUsersList получает список пользователей с фильтрами по
func (s *UserService) GetUsersList(ctx context.Context, filter model.UserFilter) ([]model.User, error) {
	users, err := s.repository.GetList(ctx, filter)
	if err != nil {
		return []model.User{}, err
	}
	s.logger.Info("Users list:")
	return users, nil
}
