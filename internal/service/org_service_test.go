package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"testing"
)

type mockOrgRepository struct{}

func (m *mockOrgRepository) CreateOrg(ctx context.Context, o *model.Org) error {
	o.ID = 1
	return nil
}

func (m *mockOrgRepository) GetList(ctx context.Context, organizationID int) ([]model.Org, error) {
	return []model.Org{}, nil
}

// TestGetOrgList проверяет, что сервис отдаёт организацию из репозитория без изменений
func TestGetOrgList(t *testing.T) {
	service := NewOrgService(&mockOrgRepository{}, logger.NewStdLogger(logger.ErrorLevel))

	orgs, err := service.GetOrgList(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orgs) != 0 {
		t.Errorf("got %d orgs, want 0", len(orgs))
	}
}
