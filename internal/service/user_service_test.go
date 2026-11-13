package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"strings"
	"testing"
)

type mockUserRepository struct {
	createErr      error
	getByIDErr     error
	getByEmailErr  error
	deleteErr      error
	listErr        error
	getByEmailUser *model.User
	admins         []int
	lastEmail      string
	created        *model.User
}

func (m *mockUserRepository) Create(ctx context.Context, u *model.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	u.ID = 1
	m.created = u
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id int) (model.User, error) {
	if m.getByIDErr != nil {
		return model.User{}, m.getByIDErr
	}
	return model.User{ID: id}, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	m.lastEmail = email
	if m.getByEmailErr != nil {
		return model.User{}, m.getByEmailErr
	}
	if m.getByEmailUser != nil {
		return *m.getByEmailUser, nil
	}
	return model.User{Email: email}, nil
}

func (m *mockUserRepository) DeleteByID(ctx context.Context, id int, organizationID *int) (model.User, error) {
	if m.deleteErr != nil {
		return model.User{}, m.deleteErr
	}
	return model.User{ID: id}, nil
}

func (m *mockUserRepository) LockActiveAdmins(ctx context.Context, organizationID int) ([]int, error) {
	return m.admins, nil
}

func (m *mockUserRepository) GetList(ctx context.Context, filter model.UserFilter) ([]model.User, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []model.User{}, nil
}

// newTestUserService собирает UserService, у которого транзакция работает с теми же моками
func newTestUserService(repo *mockUserRepository, tokens *mockRefreshTokenRepository, log logger.Logger) *UserService {
	rf := &fakeRepoFactory{user: repo, refreshToken: tokens}
	return NewUserService(repo, log, &fakeTxManager{}, rf)
}

func userFixture() model.User {
	driverID := 3
	return model.User{
		OrganizationID: 1,
		DriverID:       &driverID,
		Name:           "Ivan Petrov",
		Email:          "ivan@example.com",
		Role:           model.UserRoleDriver,
	}
}

const validPassword = "Passw0rd"

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(u *model.User)
		password string
		wantErr  error
	}{
		{name: "valid", mutate: func(u *model.User) {}, password: validPassword, wantErr: nil},
		{name: "invalid organization id", mutate: func(u *model.User) { u.OrganizationID = 0 }, password: validPassword, wantErr: model.ErrInvalidOrganizationID},
		{name: "empty name", mutate: func(u *model.User) { u.Name = "" }, password: validPassword, wantErr: model.ErrInvalidName},
		{name: "blank name", mutate: func(u *model.User) { u.Name = "   " }, password: validPassword, wantErr: model.ErrInvalidName},
		{name: "name too long", mutate: func(u *model.User) { u.Name = strings.Repeat("a", 36) }, password: validPassword, wantErr: model.ErrInvalidName},
		{name: "invalid role", mutate: func(u *model.User) { u.Role = "BOSS" }, password: validPassword, wantErr: model.ErrInvalidUserRole},
		{name: "invalid email", mutate: func(u *model.User) { u.Email = "not-an-email" }, password: validPassword, wantErr: model.ErrInvalidEmail},
		{name: "password too short", mutate: func(u *model.User) {}, password: "p1", wantErr: model.ErrForbiddenPassword},
		{name: "password same as email", mutate: func(u *model.User) { u.Email = "abc123@x.co" }, password: "abc123@x.co", wantErr: model.ErrForbiddenPassword},
		// роль DRIVER без водителя бесполезна: все её выборки ограничены своим driver_id
		{name: "driver without driver_id", mutate: func(u *model.User) { u.DriverID = nil }, password: validPassword, wantErr: model.ErrInvalidDriverID},
		{name: "dispatcher with driver_id", mutate: func(u *model.User) { u.Role = model.UserRoleDispatcher }, password: validPassword, wantErr: model.ErrInvalidDriverID},
		{name: "dispatcher without driver_id", mutate: func(u *model.User) { u.Role = model.UserRoleDispatcher; u.DriverID = nil }, password: validPassword, wantErr: nil},
	}

	repo := &mockUserRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := newTestUserService(repo, &mockRefreshTokenRepository{}, log)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := userFixture()
			tt.mutate(&u)
			_, err := svc.CreateUser(context.Background(), u, tt.password)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateUser_RepositoryError(t *testing.T) {
	repo := &mockUserRepository{createErr: model.ErrDuplicateEmail}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := newTestUserService(repo, &mockRefreshTokenRepository{}, log)

	_, err := svc.CreateUser(context.Background(), userFixture(), validPassword)
	if err != model.ErrDuplicateEmail {
		t.Errorf("got %v, want %v", err, model.ErrDuplicateEmail)
	}
}

func TestGetUserByID(t *testing.T) {
	repo := &mockUserRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := newTestUserService(repo, &mockRefreshTokenRepository{}, log)

	u, err := svc.GetUserByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 5 {
		t.Errorf("got ID %d, want 5", u.ID)
	}

	_, err = svc.GetUserByID(context.Background(), 0)
	if err != model.ErrInvalidUserID {
		t.Errorf("got %v, want %v", err, model.ErrInvalidUserID)
	}

	repo.getByIDErr = model.ErrNotFound
	_, err = svc.GetUserByID(context.Background(), 999)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestGetUserByEmail(t *testing.T) {
	repo := &mockUserRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := newTestUserService(repo, &mockRefreshTokenRepository{}, log)

	u, err := svc.GetUserByEmail(context.Background(), "ivan@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "ivan@example.com" {
		t.Errorf("got email %q, want %q", u.Email, "ivan@example.com")
	}

	_, err = svc.GetUserByEmail(context.Background(), "")
	if err != model.ErrInvalidEmail {
		t.Errorf("got %v, want %v", err, model.ErrInvalidEmail)
	}

	repo.getByEmailErr = model.ErrNotFound
	_, err = svc.GetUserByEmail(context.Background(), "missing@example.com")
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

// TestCreateUser_NormalizesEmail проверяет, что Sanya@ и sanya@ - один и тот же адрес
func TestCreateUser_NormalizesEmail(t *testing.T) {
	repo := &mockUserRepository{}
	svc := newTestUserService(repo, &mockRefreshTokenRepository{}, logger.NewStdLogger(logger.ErrorLevel))

	u := userFixture()
	u.Email = "  Sanya@Example.COM "
	if _, err := svc.CreateUser(context.Background(), u, validPassword); err != nil {
		t.Fatal(err)
	}
	if repo.created.Email != "sanya@example.com" {
		t.Errorf("stored email = %q, want sanya@example.com", repo.created.Email)
	}
}

func TestDeleteUserByID(t *testing.T) {
	admin := model.AuthContext{UserID: 1, OrganizationID: 1, Role: model.UserRoleAdmin}

	t.Run("удаление сотрудника отзывает его сессии", func(t *testing.T) {
		repo := &mockUserRepository{admins: []int{1}}
		tokens := &mockRefreshTokenRepository{}
		svc := newTestUserService(repo, tokens, logger.NewStdLogger(logger.ErrorLevel))

		u, err := svc.DeleteUserByID(context.Background(), 5, admin)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.ID != 5 {
			t.Errorf("got ID %d, want 5", u.ID)
		}
		if len(tokens.revokedUsers) != 1 || tokens.revokedUsers[0] != 5 {
			t.Errorf("refresh tokens of user 5 not revoked: %v", tokens.revokedUsers)
		}
	})

	tests := []struct {
		name    string
		id      int
		admins  []int
		repoErr error
		wantErr error
	}{
		{"некорректный id", 0, []int{1}, nil, model.ErrInvalidUserID},
		{"удалить себя", 1, []int{1, 2}, nil, model.ErrDeleteSelf},
		// после удаления 2 админов не осталось бы: 1 - это сам удаляющий, но он уже не админ,
		// например понижен в соседней транзакции, и в списке его нет
		{"последний админ", 2, []int{2}, nil, model.ErrLastAdmin},
		{"один из двух админов", 2, []int{1, 2}, nil, nil},
		{"чужой или несуществующий", 999, []int{1}, model.ErrNotFound, model.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepository{admins: tt.admins, deleteErr: tt.repoErr}
			svc := newTestUserService(repo, &mockRefreshTokenRepository{}, logger.NewStdLogger(logger.ErrorLevel))

			_, err := svc.DeleteUserByID(context.Background(), tt.id, admin)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetUsersList(t *testing.T) {
	repo := &mockUserRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := newTestUserService(repo, &mockRefreshTokenRepository{}, log)

	users, err := svc.GetUsersList(context.Background(), model.UserFilter{Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if users == nil {
		t.Errorf("expected non-nil slice")
	}

	repo.listErr = model.ErrNotFound
	_, err = svc.GetUsersList(context.Background(), model.UserFilter{Limit: 100})
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}
