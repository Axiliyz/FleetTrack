package service

import (
	"context"
	"fleettrack/internal/database"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/factory"
	"fleettrack/internal/transaction"
	"fleettrack/internal/validator"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// UserService содержит бизнес-логику пользователей
type UserService struct {
	repository  repository.UserRepository
	logger      logger.Logger
	txManager   transaction.TransactionManager
	repoFactory factory.RepositoryFactory
}

// NewUserService - конструктор сервиса юзеров
func NewUserService(r repository.UserRepository, l logger.Logger, tx transaction.TransactionManager, rf factory.RepositoryFactory) *UserService {
	return &UserService{
		repository:  r,
		logger:      l,
		txManager:   tx,
		repoFactory: rf,
	}
}

// normalizeEmail приводит почту к одному виду: Sanya@Mail.ru и sanya@mail.ru - один адрес
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// GetUserByEmail находит юзера по почте
// Возвращает юзера или ошибку
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	if len(email) < 1 {
		return model.User{}, model.ErrInvalidEmail
	}
	user, err := s.repository.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		return model.User{}, err
	}
	s.logger.Debug(fmt.Sprintf("user %d found by email", user.ID))
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
	s.logger.Debug(fmt.Sprintf("got user with id %d", id))
	return user, nil
}

// validateUser проверяет все поля юзера(кроме пароля, т.к. он зашифрован уже)
func validateUser(u model.User) error {
	if u.OrganizationID < 1 {
		return model.ErrInvalidOrganizationID
	}
	if !validator.IsNameValid(u.Name) {
		return model.ErrInvalidName
	}
	if validator.IsRoleValid(u.Role) != nil {
		return model.ErrInvalidUserRole
	}
	if validator.ValidateEmail(u.Email) != nil {
		return model.ErrInvalidEmail
	}
	// роль DRIVER без водителя бесполезна: все её выборки ограничены своим driver_id.
	// И наоборот, водитель у диспетчера ничего бы не значил
	if (u.Role == model.UserRoleDriver) != (u.DriverID != nil) {
		return model.ErrInvalidDriverID
	}
	if u.DriverID != nil && *u.DriverID < 1 {
		return model.ErrInvalidDriverID
	}
	return nil
}

// CreateUser создаёт пользователя
// Возвращает ошибку при неудаче
func (s *UserService) CreateUser(ctx context.Context, u model.User, password string) (model.User, error) {
	u.Email = normalizeEmail(u.Email)
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

// DeleteUserByID мягко удаляет пользователя организации actor и отзывает все его refresh-токены.
// Себя удалить нельзя (model.ErrDeleteSelf), последнего админа - тоже (model.ErrLastAdmin):
// организация без админа никого больше не заведёт.
// Чужой пользователь неотличим от несуществующего - model.ErrNotFound
func (s *UserService) DeleteUserByID(ctx context.Context, id int, actor model.AuthContext) (model.User, error) {
	if id <= 0 {
		return model.User{}, model.ErrInvalidUserID
	}
	if id == actor.UserID {
		return model.User{}, model.ErrDeleteSelf
	}

	var deleted model.User
	err := s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)

		// блокировка админов идёт первой: так удаление одного админа ждёт удаления другого
		admins, err := repos.User.LockActiveAdmins(ctx, actor.OrganizationID)
		if err != nil {
			return err
		}
		if len(admins) == 1 && slices.Contains(admins, id) {
			return model.ErrLastAdmin
		}

		orgID := actor.OrganizationID
		if deleted, err = repos.User.DeleteByID(ctx, id, &orgID); err != nil {
			return err
		}
		return repos.RefreshToken.RevokeAllForUser(ctx, id)
	})
	if err != nil {
		return model.User{}, err
	}
	s.logger.Info(fmt.Sprintf("user %d deleted by user %d", id, actor.UserID))
	return deleted, nil
}

// GetUsersList получает список пользователей с фильтрами по
func (s *UserService) GetUsersList(ctx context.Context, filter model.UserFilter) ([]model.User, error) {
	users, err := s.repository.GetList(ctx, filter)
	if err != nil {
		return []model.User{}, err
	}
	s.logger.Debug("got users list")
	return users, nil
}
