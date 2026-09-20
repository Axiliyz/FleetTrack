package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fleettrack/internal/repository/factory"
	"fleettrack/internal/transaction"
	"fleettrack/internal/validator"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepository         repository.UserRepository
	refreshTokenRepository repository.RefreshTokenRepository
	jwtService             *JWTService
	refreshTTL             time.Duration
	logger                 logger.Logger
	txManager              transaction.TransactionManager
	repoFactory            factory.RepositoryFactory
}

func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtService *JWTService,
	refreshTTL time.Duration,
	l logger.Logger,
	tx transaction.TransactionManager,
	repoFactory factory.RepositoryFactory,
) *AuthService {
	return &AuthService{
		userRepository:         userRepo,
		refreshTokenRepository: refreshTokenRepo,
		jwtService:             jwtService,
		refreshTTL:             refreshTTL,
		logger:                 l,
		txManager:              tx,
		repoFactory:            repoFactory,
	}
}

// issueTokens генерирует access-токен и новый refresh-токен для юзера,
// сохраняя хеш refresh-токена в БД. Возвращает оба токена клиенту -
// сам plain refresh-токен на сервере не хранится, только его хеш.
func (s *AuthService) issueTokens(ctx context.Context, user model.User) (model.AuthResult, error) {
	accessToken, err := s.jwtService.Generate(user)
	if err != nil {
		return model.AuthResult{}, err
	}

	plainRefresh, hash, err := generateRefreshToken()
	if err != nil {
		return model.AuthResult{}, err
	}

	rt := model.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}
	if err := s.refreshTokenRepository.Create(ctx, &rt); err != nil {
		return model.AuthResult{}, err
	}

	return model.AuthResult{AccessToken: accessToken, RefreshToken: plainRefresh}, nil
}

func (s *AuthService) Login(ctx context.Context, email string, password string) (model.AuthResult, error) {
	user, err := s.userRepository.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.AuthResult{}, model.ErrInvalidCredentials
		}
		return model.AuthResult{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return model.AuthResult{}, model.ErrInvalidCredentials
	}
	return s.issueTokens(ctx, user)
}

func (s *AuthService) RegisterCompany(ctx context.Context, orgName, adminName, adminEmail, adminPassword string) (model.AuthResult, error) {
	if strings.TrimSpace(orgName) == "" {
		return model.AuthResult{}, model.ErrInvalidOrgName
	}
	if err := validator.ValidatePassword(adminPassword); err != nil {
		return model.AuthResult{}, err
	}
	if err := validator.CheckPasswordLegit(adminPassword, adminEmail, adminName); err != nil {
		return model.AuthResult{}, err
	}

	admin := model.User{Name: adminName, Email: adminEmail, Role: model.UserRoleAdmin}

	err := s.txManager.WithTx(ctx, func(tx database.DBTX) error {
		repos := s.repoFactory.New(tx)

		org := model.Org{Name: orgName}
		if err := repos.Org.CreateOrg(ctx, &org); err != nil {
			return err
		}
		admin.OrganizationID = org.ID

		if err := validateUser(admin); err != nil {
			return err
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		admin.PasswordHash = string(hash)
		return repos.User.Create(ctx, &admin)
	})
	if err != nil {
		return model.AuthResult{}, err
	}

	return s.issueTokens(ctx, admin)
}

// Refresh проверяет refresh-токен, отзывает его (ротация) и выдаёт новую пару токенов.
// Если токен не найден, уже отозван или истёк - возвращает model.ErrInvalidToken.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (model.AuthResult, error) {
	stored, err := s.refreshTokenRepository.RevokeActiveByHash(ctx, hashRefreshToken(refreshToken))
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.AuthResult{}, model.ErrInvalidToken
		}
		return model.AuthResult{}, err
	}

	user, err := s.userRepository.GetByID(ctx, stored.UserID)
	if err != nil {
		return model.AuthResult{}, err
	}

	return s.issueTokens(ctx, user)
}

// Logout отзывает присланный refresh-токен. Идемпотентен: если токен уже
// не существует/отозван/истёк, ошибку не возвращает - выйти можно и так.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	stored, err := s.refreshTokenRepository.GetActiveByHash(ctx, hashRefreshToken(refreshToken))
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil
		}
		return err
	}
	return s.refreshTokenRepository.Revoke(ctx, stored.ID)
}

func generateRefreshToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	return plain, hashRefreshToken(plain), nil
}

func hashRefreshToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
