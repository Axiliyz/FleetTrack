package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const testRefreshTTL = 30 * 24 * time.Hour

// mockRefreshTokenRepository - мок repository.RefreshTokenRepository
type mockRefreshTokenRepository struct {
	createErr      error
	getActiveErr   error
	getActiveToken *model.RefreshToken
	revokeErr      error
	revokedIDs     []int
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, t *model.RefreshToken) error {
	if m.createErr != nil {
		return m.createErr
	}
	t.ID = 1
	return nil
}

func (m *mockRefreshTokenRepository) GetActiveByHash(ctx context.Context, hash string) (model.RefreshToken, error) {
	if m.getActiveErr != nil {
		return model.RefreshToken{}, m.getActiveErr
	}
	if m.getActiveToken != nil {
		return *m.getActiveToken, nil
	}
	return model.RefreshToken{}, model.ErrNotFound
}

func (m *mockRefreshTokenRepository) Revoke(ctx context.Context, id int) error {
	m.revokedIDs = append(m.revokedIDs, id)
	return m.revokeErr
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}
	return string(hash)
}

func TestAuthService_Login_Success(t *testing.T) {
	const password = "Passw0rd"
	user := model.User{ID: 7, Email: "ivan@example.com", Role: model.UserRoleDispatcher, PasswordHash: hashPassword(t, password)}

	repo := &mockUserRepository{getByEmailUser: &user}
	jwtService := NewJWTService("test-secret", time.Minute)
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewAuthService(repo, &mockRefreshTokenRepository{}, jwtService, testRefreshTTL, log, &mockTxManager{}, &mockRepoFactory{})

	result, err := svc.Login(context.Background(), user.Email, password)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if result.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}

	claims, err := jwtService.Parse(result.AccessToken)
	if err != nil {
		t.Fatalf("unexpected error parsing issued token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("got UserID %d, want %d", claims.UserID, user.ID)
	}
	if claims.Role != user.Role {
		t.Errorf("got Role %q, want %q", claims.Role, user.Role)
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := &mockUserRepository{getByEmailErr: model.ErrNotFound}
	jwtService := NewJWTService("test-secret", time.Minute)
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewAuthService(repo, &mockRefreshTokenRepository{}, jwtService, testRefreshTTL, log, &mockTxManager{}, &mockRepoFactory{})

	_, err := svc.Login(context.Background(), "missing@example.com", "whatever")
	if err != model.ErrInvalidCredentials {
		t.Errorf("got %v, want %v", err, model.ErrInvalidCredentials)
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	user := model.User{ID: 7, Email: "ivan@example.com", Role: model.UserRoleDriver, PasswordHash: hashPassword(t, "correct-password")}

	repo := &mockUserRepository{getByEmailUser: &user}
	jwtService := NewJWTService("test-secret", time.Minute)
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewAuthService(repo, &mockRefreshTokenRepository{}, jwtService, testRefreshTTL, log, &mockTxManager{}, &mockRepoFactory{})

	_, err := svc.Login(context.Background(), user.Email, "wrong-password")
	if err != model.ErrInvalidCredentials {
		t.Errorf("got %v, want %v", err, model.ErrInvalidCredentials)
	}
}

func TestAuthService_Login_RepositoryError(t *testing.T) {
	repo := &mockUserRepository{getByEmailErr: model.ErrConnectingDB}
	jwtService := NewJWTService("test-secret", time.Minute)
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewAuthService(repo, &mockRefreshTokenRepository{}, jwtService, testRefreshTTL, log, &mockTxManager{}, &mockRepoFactory{})

	_, err := svc.Login(context.Background(), "ivan@example.com", "whatever")
	if err != model.ErrConnectingDB {
		t.Errorf("got %v, want %v (non-not-found repository errors should propagate as-is)", err, model.ErrConnectingDB)
	}
}

func TestAuthService_RegisterCompany_Success(t *testing.T) {
	orgRepo := &mockOrgRepository{}
	userRepo := &mockUserRepository{}
	jwtService := NewJWTService("test-secret", time.Minute)
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewAuthService(nil, &mockRefreshTokenRepository{}, jwtService, testRefreshTTL, log, &mockTxManager{}, &mockRepoFactory{orgRepo: orgRepo, userRepo: userRepo})

	result, err := svc.RegisterCompany(context.Background(), "Alpha Fleet", "Root Admin", "admin@example.com", "AdminPass1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if result.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}

	claims, err := jwtService.Parse(result.AccessToken)
	if err != nil {
		t.Fatalf("unexpected error parsing issued token: %v", err)
	}
	if claims.Role != model.UserRoleAdmin {
		t.Errorf("got Role %q, want %q", claims.Role, model.UserRoleAdmin)
	}
	if claims.OrganizationID != 1 {
		t.Errorf("got OrganizationID %d, want 1 (from mockOrgRepository)", claims.OrganizationID)
	}
}

func TestAuthService_RegisterCompany_EmptyOrgName(t *testing.T) {
	svc := NewAuthService(nil, &mockRefreshTokenRepository{}, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{})

	_, err := svc.RegisterCompany(context.Background(), "   ", "Root Admin", "admin@example.com", "AdminPass1")
	if err != model.ErrInvalidOrgName {
		t.Errorf("got %v, want %v", err, model.ErrInvalidOrgName)
	}
}

func TestAuthService_RegisterCompany_InvalidPassword(t *testing.T) {
	svc := NewAuthService(nil, &mockRefreshTokenRepository{}, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{})

	_, err := svc.RegisterCompany(context.Background(), "Alpha Fleet", "Root Admin", "admin@example.com", "short")
	if err != model.ErrForbiddenPassword {
		t.Errorf("got %v, want %v", err, model.ErrForbiddenPassword)
	}
}

func TestAuthService_RegisterCompany_DuplicateEmail(t *testing.T) {
	orgRepo := &mockOrgRepository{}
	userRepo := &mockUserRepository{createErr: model.ErrDuplicateEmail}
	svc := NewAuthService(nil, &mockRefreshTokenRepository{}, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{orgRepo: orgRepo, userRepo: userRepo})

	_, err := svc.RegisterCompany(context.Background(), "Alpha Fleet", "Root Admin", "admin@example.com", "AdminPass1")
	if err != model.ErrDuplicateEmail {
		t.Errorf("got %v, want %v", err, model.ErrDuplicateEmail)
	}
}

func TestAuthService_Refresh_Success(t *testing.T) {
	user := model.User{ID: 7, Role: model.UserRoleDispatcher}
	userRepo := &mockUserRepository{getByIDErr: nil}
	refreshRepo := &mockRefreshTokenRepository{getActiveToken: &model.RefreshToken{ID: 42, UserID: user.ID}}
	jwtService := NewJWTService("test-secret", time.Minute)
	svc := NewAuthService(userRepo, refreshRepo, jwtService, testRefreshTTL, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{})

	result, err := svc.Refresh(context.Background(), "some-refresh-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatal("expected non-empty access and refresh tokens")
	}
	if len(refreshRepo.revokedIDs) != 1 || refreshRepo.revokedIDs[0] != 42 {
		t.Errorf("expected old token (id=42) to be revoked, got revoked ids %v", refreshRepo.revokedIDs)
	}
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{}
	svc := NewAuthService(&mockUserRepository{}, refreshRepo, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{})

	_, err := svc.Refresh(context.Background(), "unknown-token")
	if err != model.ErrInvalidToken {
		t.Errorf("got %v, want %v", err, model.ErrInvalidToken)
	}
}

func TestAuthService_Logout_RevokesToken(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{getActiveToken: &model.RefreshToken{ID: 5, UserID: 1}}
	svc := NewAuthService(&mockUserRepository{}, refreshRepo, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{})

	if err := svc.Logout(context.Background(), "some-refresh-token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refreshRepo.revokedIDs) != 1 || refreshRepo.revokedIDs[0] != 5 {
		t.Errorf("expected token (id=5) to be revoked, got revoked ids %v", refreshRepo.revokedIDs)
	}
}

func TestAuthService_Logout_UnknownTokenIsIdempotent(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{}
	svc := NewAuthService(&mockUserRepository{}, refreshRepo, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{})

	if err := svc.Logout(context.Background(), "unknown-token"); err != nil {
		t.Errorf("logout of unknown token should not error, got %v", err)
	}
}
