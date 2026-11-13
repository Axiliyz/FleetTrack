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
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TokenIssuer выпускает access-токен пользователю; реализует *JWTService
type TokenIssuer interface {
	Generate(user model.User) (string, error)
}

// dummyPasswordHash - bcrypt-хеш, с которым сравнивается пароль, если почта не найдена.
// Без него неизвестная почта отвечала бы мгновенно, а известная - через 50-100 мс bcrypt,
// и по времени ответа можно было бы перебрать, кто зарегистрирован
var dummyPasswordHash = sync.OnceValue(func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("fleettrack-dummy-password"), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Sprintf("bcrypt dummy hash: %s", err))
	}
	return hash
})

// AuthService реализует бизнес-логику логина, регистрации организации и ротации refresh-токенов
type AuthService struct {
	userRepository         repository.UserRepository
	refreshTokenRepository repository.RefreshTokenRepository
	jwtService             TokenIssuer
	refreshTTL             time.Duration
	logger                 logger.Logger
	txManager              transaction.TransactionManager
	repoFactory            factory.RepositoryFactory
}

// NewAuthService собирает AuthService из репозиториев, JWT-сервиса и менеджера транзакций
func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtService TokenIssuer,
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
// familyID - семья, в которой продолжается ротация; пустая строка начинает новую (логин)
func (s *AuthService) issueTokens(ctx context.Context, user model.User, familyID string) (model.AuthResult, error) {
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
		FamilyID:  familyID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}
	if err := s.refreshTokenRepository.Create(ctx, &rt); err != nil {
		return model.AuthResult{}, err
	}

	return model.AuthResult{AccessToken: accessToken, RefreshToken: plainRefresh}, nil
}

// Login проверяет email и пароль пользователя и выдаёт пару access/refresh токенов.
// Неизвестная почта и неверный пароль неотличимы ни по ответу, ни по времени
func (s *AuthService) Login(ctx context.Context, email string, password string) (model.AuthResult, error) {
	user, err := s.userRepository.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash(), []byte(password))
			return model.AuthResult{}, model.ErrInvalidCredentials
		}
		return model.AuthResult{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return model.AuthResult{}, model.ErrInvalidCredentials
	}
	return s.issueTokens(ctx, user, "")
}

// RegisterOrganization создаёт организацию и её первичного администратора в одной транзакции и выдаёт токены
func (s *AuthService) RegisterOrganization(ctx context.Context, orgName, adminName, adminEmail, adminPassword string) (model.AuthResult, error) {
	if strings.TrimSpace(orgName) == "" {
		return model.AuthResult{}, model.ErrInvalidOrgName
	}
	if err := validator.ValidatePassword(adminPassword); err != nil {
		return model.AuthResult{}, err
	}
	if err := validator.CheckPasswordLegit(adminPassword, adminEmail, adminName); err != nil {
		return model.AuthResult{}, err
	}

	admin := model.User{Name: adminName, Email: normalizeEmail(adminEmail), Role: model.UserRoleAdmin}

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

	return s.issueTokens(ctx, admin, "")
}

// Refresh проверяет refresh-токен, отзывает его (ротация) и выдаёт новую пару токенов той же семьи.
// Если токен не найден, истёк или пользователь удалён - возвращает model.ErrInvalidToken.
//
// Если пришёл уже отозванный, но ещё не истёкший токен, его предъявляют второй раз:
// значит, токен украли, и неизвестно, кто из двоих - владелец. Отзывается вся семья,
// обоим придётся войти заново. Ошибка для клиента та же - model.ErrInvalidToken
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (model.AuthResult, error) {
	hash := hashRefreshToken(refreshToken)
	stored, err := s.refreshTokenRepository.RevokeActiveByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.AuthResult{}, s.checkReuse(ctx, hash)
		}
		return model.AuthResult{}, err
	}

	user, err := s.userRepository.GetByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.AuthResult{}, model.ErrInvalidToken
		}
		return model.AuthResult{}, err
	}

	return s.issueTokens(ctx, user, stored.FamilyID)
}

// checkReuse разбирается, почему токен не прошёл ротацию. Повторное предъявление
// отозванного токена отзывает его семью. Всегда возвращает ошибку для ответа клиенту
func (s *AuthService) checkReuse(ctx context.Context, hash string) error {
	stored, err := s.refreshTokenRepository.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.ErrInvalidToken
		}
		return err
	}
	if stored.RevokedAt == nil || !stored.ExpiresAt.After(time.Now()) {
		return model.ErrInvalidToken
	}
	if err := s.refreshTokenRepository.RevokeFamily(ctx, stored.FamilyID); err != nil {
		return err
	}
	s.logger.Warn(fmt.Sprintf("%s: user %d, token family %s revoked", model.ErrTokenReused, stored.UserID, stored.FamilyID))
	return model.ErrInvalidToken
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
