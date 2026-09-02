package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fmt"

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

// CreateUser создаёт пользователя
// Возвращает ошибку при неудаче
func (s *UserService) CreateUser(ctx context.Context, u model.User, password string) error {
	if err := validateUser(u) != nil; err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	if err := s.repository.Create(ctx, &u); err != nil {
		return err
	}
	message := fmt.Sprintf("created user with id: %d", u.ID)
	s.logger.Info(message)
	return nil
}

// DeleteUserByID удаляет пользователя по его ID
// Возвращает удалённого юзера или ошибку
func (s UserService) DeleteUserByID(ctx context.Context, id int) (model.User, error) {
	if id <= 0 {
		return model.User{}, model.ErrInvalidUserID
	}

	user, err := s.repository.DeleteByID(ctx, id)
	if err != nil {
		return model.User{}, err
	}
	message := fmt.Sprintf("user with id %d deleted", id)
	s.logger.Info(message)
	return user, nil
}
