package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fmt"
	"strings"
)

// OrgService реализует бизнес-логику работы с организациями.
type OrgService struct {
	repository repository.OrgRepository
	logger     logger.Logger
}

// NewOrgService создаёт новый сервис организаций.
func NewOrgService(r repository.OrgRepository, l logger.Logger) *OrgService {
	return &OrgService{
		repository: r,
		logger:     l,
	}
}

// CreateOrg валидирует и сохраняет новую организацию.
func (s *OrgService) CreateOrg(ctx context.Context, o model.Org) (model.Org, error) {
	if strings.TrimSpace(o.Name) == "" {
		return model.Org{}, model.ErrInvalidOrgName
	}

	if err := s.repository.CreateOrg(ctx, &o); err != nil {
		return model.Org{}, err
	}

	s.logger.Info(fmt.Sprintf("Organization %d created", o.ID))
	return o, nil
}
