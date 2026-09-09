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
}

func (m *mockUserRepository) Create(ctx context.Context, u *model.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	u.ID = 1
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id int) (model.User, error) {
	if m.getByIDErr != nil {
		return model.User{}, m.getByIDErr
	}
	return model.User{ID: id}, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
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

func (m *mockUserRepository) GetList(ctx context.Context, filter model.UserFilter) ([]model.User, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []model.User{}, nil
}

func userFixture() model.User {
	return model.User{
		OrganizationID: 1,
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
	}

	repo := &mockUserRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewUserService(repo, log)

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
	svc := NewUserService(repo, log)

	_, err := svc.CreateUser(context.Background(), userFixture(), validPassword)
	if err != model.ErrDuplicateEmail {
		t.Errorf("got %v, want %v", err, model.ErrDuplicateEmail)
	}
}

func TestGetUserByID(t *testing.T) {
	repo := &mockUserRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewUserService(repo, log)

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
	svc := NewUserService(repo, log)

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

func TestDeleteUserByID(t *testing.T) {
	repo := &mockUserRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewUserService(repo, log)

	u, err := svc.DeleteUserByID(context.Background(), 5, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 5 {
		t.Errorf("got ID %d, want 5", u.ID)
	}

	_, err = svc.DeleteUserByID(context.Background(), 0, nil)
	if err != model.ErrInvalidUserID {
		t.Errorf("got %v, want %v", err, model.ErrInvalidUserID)
	}

	repo.deleteErr = model.ErrNotFound
	_, err = svc.DeleteUserByID(context.Background(), 999, nil)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestGetUsersList(t *testing.T) {
	repo := &mockUserRepository{}
	log := logger.NewStdLogger(logger.DebugLevel)
	svc := NewUserService(repo, log)

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
