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
	// byHash - токен в любом состоянии, который вернёт GetByHash
	byHash          *model.RefreshToken
	revokedFamilies []string
	revokedUsers    []int
	created         []model.RefreshToken
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, t *model.RefreshToken) error {
	if m.createErr != nil {
		return m.createErr
	}
	t.ID = 1
	if t.FamilyID == "" {
		t.FamilyID = "new-family"
	}
	m.created = append(m.created, *t)
	return nil
}

func (m *mockRefreshTokenRepository) GetByHash(ctx context.Context, hash string) (model.RefreshToken, error) {
	if m.byHash != nil {
		return *m.byHash, nil
	}
	return model.RefreshToken{}, model.ErrNotFound
}

func (m *mockRefreshTokenRepository) RevokeFamily(ctx context.Context, familyID string) error {
	m.revokedFamilies = append(m.revokedFamilies, familyID)
	return nil
}

func (m *mockRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID int) error {
	m.revokedUsers = append(m.revokedUsers, userID)
	return nil
}

func (m *mockRefreshTokenRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
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

func (m *mockRefreshTokenRepository) RevokeActiveByHash(ctx context.Context, hash string) (model.RefreshToken, error) {
	if m.getActiveErr != nil {
		return model.RefreshToken{}, m.getActiveErr
	}
	if m.getActiveToken != nil {
		m.revokedIDs = append(m.revokedIDs, m.getActiveToken.ID)
		return *m.getActiveToken, nil
	}
	return model.RefreshToken{}, model.ErrNotFound
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

func TestAuthService_RegisterOrganization_Success(t *testing.T) {
	orgRepo := &mockOrgRepository{}
	userRepo := &mockUserRepository{}
	jwtService := NewJWTService("test-secret", time.Minute)
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewAuthService(nil, &mockRefreshTokenRepository{}, jwtService, testRefreshTTL, log, &mockTxManager{}, &mockRepoFactory{orgRepo: orgRepo, userRepo: userRepo})

	result, err := svc.RegisterOrganization(context.Background(), "Alpha Fleet", "Root Admin", "admin@example.com", "AdminPass1")
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

func TestAuthService_RegisterOrganization_EmptyOrgName(t *testing.T) {
	svc := NewAuthService(nil, &mockRefreshTokenRepository{}, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{})

	_, err := svc.RegisterOrganization(context.Background(), "   ", "Root Admin", "admin@example.com", "AdminPass1")
	if err != model.ErrInvalidOrgName {
		t.Errorf("got %v, want %v", err, model.ErrInvalidOrgName)
	}
}

func TestAuthService_RegisterOrganization_InvalidPassword(t *testing.T) {
	svc := NewAuthService(nil, &mockRefreshTokenRepository{}, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{})

	_, err := svc.RegisterOrganization(context.Background(), "Alpha Fleet", "Root Admin", "admin@example.com", "short")
	if err != model.ErrForbiddenPassword {
		t.Errorf("got %v, want %v", err, model.ErrForbiddenPassword)
	}
}

func TestAuthService_RegisterOrganization_DuplicateEmail(t *testing.T) {
	orgRepo := &mockOrgRepository{}
	userRepo := &mockUserRepository{createErr: model.ErrDuplicateEmail}
	svc := NewAuthService(nil, &mockRefreshTokenRepository{}, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.DebugLevel), &mockTxManager{}, &mockRepoFactory{orgRepo: orgRepo, userRepo: userRepo})

	_, err := svc.RegisterOrganization(context.Background(), "Alpha Fleet", "Root Admin", "admin@example.com", "AdminPass1")
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

// TestAuthService_Refresh_KeepsFamily проверяет, что ротация продолжает семью токена, а не начинает новую
func TestAuthService_Refresh_KeepsFamily(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{getActiveToken: &model.RefreshToken{ID: 42, UserID: 7, FamilyID: "fam-1"}}
	svc := NewAuthService(&mockUserRepository{}, refreshRepo, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.ErrorLevel), &mockTxManager{}, &mockRepoFactory{})

	if _, err := svc.Refresh(context.Background(), "token"); err != nil {
		t.Fatal(err)
	}
	if len(refreshRepo.created) != 1 || refreshRepo.created[0].FamilyID != "fam-1" {
		t.Errorf("new token family = %+v, want fam-1", refreshRepo.created)
	}
}

// TestAuthService_Refresh_ReuseRevokesFamily проверяет детект повторного использования:
// отозванный, но не истёкший токен пришёл снова - значит, его украли, и отзывается вся семья
func TestAuthService_Refresh_ReuseRevokesFamily(t *testing.T) {
	revokedAt := time.Now().Add(-time.Minute)
	refreshRepo := &mockRefreshTokenRepository{byHash: &model.RefreshToken{
		ID: 42, UserID: 7, FamilyID: "fam-1", RevokedAt: &revokedAt, ExpiresAt: time.Now().Add(time.Hour),
	}}
	svc := NewAuthService(&mockUserRepository{}, refreshRepo, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.ErrorLevel), &mockTxManager{}, &mockRepoFactory{})

	_, err := svc.Refresh(context.Background(), "stolen-token")
	if err != model.ErrInvalidToken {
		t.Errorf("got %v, want %v", err, model.ErrInvalidToken)
	}
	if len(refreshRepo.revokedFamilies) != 1 || refreshRepo.revokedFamilies[0] != "fam-1" {
		t.Errorf("family not revoked: %v", refreshRepo.revokedFamilies)
	}
	if len(refreshRepo.created) != 0 {
		t.Error("по украденному токену не должны выдаваться новые")
	}
}

// TestAuthService_Refresh_ExpiredIsNotReuse проверяет, что истёкший токен - просто недействителен,
// семью он не трогает: его мог предъявить и сам владелец после долгого перерыва
func TestAuthService_Refresh_ExpiredIsNotReuse(t *testing.T) {
	revokedAt := time.Now().Add(-48 * time.Hour)
	refreshRepo := &mockRefreshTokenRepository{byHash: &model.RefreshToken{
		ID: 42, FamilyID: "fam-1", RevokedAt: &revokedAt, ExpiresAt: time.Now().Add(-time.Hour),
	}}
	svc := NewAuthService(&mockUserRepository{}, refreshRepo, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.ErrorLevel), &mockTxManager{}, &mockRepoFactory{})

	if _, err := svc.Refresh(context.Background(), "old-token"); err != model.ErrInvalidToken {
		t.Errorf("got %v, want %v", err, model.ErrInvalidToken)
	}
	if len(refreshRepo.revokedFamilies) != 0 {
		t.Errorf("family revoked for an expired token: %v", refreshRepo.revokedFamilies)
	}
}

// TestAuthService_Refresh_DeletedUser проверяет, что токен удалённого пользователя не обновляется
func TestAuthService_Refresh_DeletedUser(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{getActiveToken: &model.RefreshToken{ID: 42, UserID: 7}}
	userRepo := &mockUserRepository{getByIDErr: model.ErrNotFound}
	svc := NewAuthService(userRepo, refreshRepo, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.ErrorLevel), &mockTxManager{}, &mockRepoFactory{})

	if _, err := svc.Refresh(context.Background(), "token"); err != model.ErrInvalidToken {
		t.Errorf("got %v, want %v", err, model.ErrInvalidToken)
	}
}

// TestAuthService_Login_NormalizesEmail проверяет, что вход не зависит от регистра почты
func TestAuthService_Login_NormalizesEmail(t *testing.T) {
	userRepo := &mockUserRepository{getByEmailErr: model.ErrNotFound}
	svc := NewAuthService(userRepo, &mockRefreshTokenRepository{}, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.ErrorLevel), &mockTxManager{}, &mockRepoFactory{})

	_, _ = svc.Login(context.Background(), " Ivan@Example.COM ", "whatever")
	if userRepo.lastEmail != "ivan@example.com" {
		t.Errorf("looked up %q, want ivan@example.com", userRepo.lastEmail)
	}
}

// TestAuthService_Login_UnknownEmailSpendsBcrypt проверяет, что неизвестная почта отвечает не мгновенно:
// без сравнения с фиктивным хешем по времени ответа видно, какие почты зарегистрированы.
// Нижняя граница с большим запасом: bcrypt с DefaultCost занимает десятки миллисекунд
func TestAuthService_Login_UnknownEmailSpendsBcrypt(t *testing.T) {
	userRepo := &mockUserRepository{getByEmailErr: model.ErrNotFound}
	svc := NewAuthService(userRepo, &mockRefreshTokenRepository{}, NewJWTService("test-secret", time.Minute), testRefreshTTL, logger.NewStdLogger(logger.ErrorLevel), &mockTxManager{}, &mockRepoFactory{})
	dummyPasswordHash() // хеш считается один раз при первом входе, в замер это не должно попасть

	start := time.Now()
	_, err := svc.Login(context.Background(), "nobody@example.com", "whatever")
	elapsed := time.Since(start)

	if err != model.ErrInvalidCredentials {
		t.Errorf("got %v, want %v", err, model.ErrInvalidCredentials)
	}
	if elapsed < 5*time.Millisecond {
		t.Errorf("unknown email answered in %s - bcrypt не выполнялся", elapsed)
	}
}
