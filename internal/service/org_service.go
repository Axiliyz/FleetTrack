package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
)

// OrgService реализует бизнес-логику работы с организациями.
// Организация создаётся только регистрацией (AuthService.RegisterOrganization) - сразу с админом:
// пустая организация без пользователей никому не нужна
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

// GetOrgList возвращает организацию с данным ID в виде списка из одного элемента -
// каждый юзер видит только свою организацию.
func (s *OrgService) GetOrgList(ctx context.Context, organizationID int) ([]model.Org, error) {
	orgs, err := s.repository.GetList(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	s.logger.Debug("got organizations list")
	return orgs, nil
}
