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

func (m *mockOrgRepository) GetList(ctx context.Context) ([]model.Org, error) {
	return []model.Org{}, nil
}

func TestCreateOrg(t *testing.T) {
	tests := []struct {
		name    string
		org     model.Org
		wantErr error
	}{
		{
			name:    "valid",
			org:     model.Org{Name: "Acme"},
			wantErr: nil,
		},
		{
			name:    "empty name",
			org:     model.Org{Name: ""},
			wantErr: model.ErrInvalidOrgName,
		},
		{
			name:    "blank name",
			org:     model.Org{Name: "   "},
			wantErr: model.ErrInvalidOrgName,
		},
	}

	repo := &mockOrgRepository{}
	logger := logger.NewStdLogger(logger.DebugLevel)
	service := NewOrgService(repo, logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateOrg(context.Background(), tt.org)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}
