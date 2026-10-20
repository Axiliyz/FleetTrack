package service

import (
	"context"
	"fleettrack/internal/model"
	"fleettrack/internal/repository"
	"fmt"
	"time"
)

const (
	// snapshotShiftWindow — за какой период в снимок попадают завершённые рейсы и закрытые алерты
	snapshotShiftWindow = 24 * time.Hour
	// snapshotMaxRows — потолок строк на каждый список. Парку больше этого нужна пагинация снимка,
	// а не один ответ на мегабайты
	snapshotMaxRows = 5000
)

// Узкие интерфейсы: снимку нужен ровно один метод от каждого источника,
// и тестам не приходится реализовывать целые репозитории ради одного списка

type snapshotVehicles interface {
	GetList(ctx context.Context, filter model.VehicleFilter) ([]model.Vehicle, error)
}

type snapshotDrivers interface {
	GetList(ctx context.Context, filter model.DriverFilter) ([]model.Driver, error)
}

type snapshotRules interface {
	GetList(ctx context.Context, filter model.AlertRuleFilter) ([]model.AlertRule, error)
}

type snapshotOrgs interface {
	GetList(ctx context.Context, organizationID int) ([]model.Org, error)
}

// FleetService собирает снимок парка для карты диспетчера
type FleetService struct {
	vehicles snapshotVehicles
	drivers  snapshotDrivers
	rules    snapshotRules
	orgs     snapshotOrgs
	snapshot repository.SnapshotRepository
	now      func() time.Time
}

// NewFleetService создаёт сервис снимка парка
func NewFleetService(
	v snapshotVehicles, d snapshotDrivers, r snapshotRules, o snapshotOrgs, s repository.SnapshotRepository,
) *FleetService {
	return &FleetService{vehicles: v, drivers: d, rules: r, orgs: o, snapshot: s, now: time.Now}
}

// Snapshot возвращает состояние парка организации actor.
// Запросы идут друг за другом без общей транзакции: между ними могут успеть прийти новые точки,
// и это не страшно — поток событий догонит карту сразу после подключения.
// Ошибки: model.ErrForbidden — роли не положен обзор всего парка (водитель видит только свои рейсы)
func (s *FleetService) Snapshot(ctx context.Context, actor model.AuthContext) (model.FleetSnapshot, error) {
	if !canViewFleet(actor.Role) {
		return model.FleetSnapshot{}, fmt.Errorf("%w: role %s can't view fleet snapshot", model.ErrForbidden, actor.Role)
	}
	orgID := actor.OrganizationID
	since := s.now().Add(-snapshotShiftWindow)
	limit := snapshotMaxRows

	orgs, err := s.orgs.GetList(ctx, orgID)
	if err != nil {
		return model.FleetSnapshot{}, fmt.Errorf("organization: %w", err)
	}
	if len(orgs) == 0 {
		return model.FleetSnapshot{}, model.ErrNotFound
	}

	var snap model.FleetSnapshot
	snap.Org = orgs[0]

	if snap.Vehicles, err = s.vehicles.GetList(ctx, model.VehicleFilter{OrganizationID: &orgID, Limit: limit}); err != nil {
		return model.FleetSnapshot{}, fmt.Errorf("vehicles: %w", err)
	}
	if snap.Drivers, err = s.drivers.GetList(ctx, model.DriverFilter{OrganizationID: &orgID, Limit: limit}); err != nil {
		return model.FleetSnapshot{}, fmt.Errorf("drivers: %w", err)
	}
	if snap.Rules, err = s.rules.GetList(ctx, model.AlertRuleFilter{OrganizationID: &orgID, Limit: &limit}); err != nil {
		return model.FleetSnapshot{}, fmt.Errorf("alert rules: %w", err)
	}
	if snap.Trips, err = s.snapshot.ShiftTrips(ctx, orgID, since, limit); err != nil {
		return model.FleetSnapshot{}, fmt.Errorf("trips: %w", err)
	}
	if snap.Alerts, err = s.snapshot.ShiftAlerts(ctx, orgID, since, limit); err != nil {
		return model.FleetSnapshot{}, fmt.Errorf("alerts: %w", err)
	}
	if snap.Positions, err = s.snapshot.LatestPositions(ctx, orgID, limit); err != nil {
		return model.FleetSnapshot{}, fmt.Errorf("positions: %w", err)
	}
	return snap, nil
}

// canViewFleet — кому положен обзор всего парка организации
func canViewFleet(role model.UserRole) bool {
	switch role {
	case model.UserRoleAdmin, model.UserRoleDispatcher, model.UserRoleAnalytic:
		return true
	default:
		return false
	}
}
