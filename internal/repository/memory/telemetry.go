package memory

import (
	"context"
	"fleettrack/internal/model"
)

// MemoryTelemetryRepository позволяет сохранять данные в память
type MemoryTelemetryRepository struct {
	telemetry map[int]model.Telemetry
	byVehicle map[int][]model.Telemetry
	current   map[int]model.Telemetry
	nextID    int
}

// NewMemoryTelemetryRepository создаёт новый репозиторий для сохранения в память
func NewMemoryTelemetryRepository() *MemoryTelemetryRepository {
	return &MemoryTelemetryRepository{
		telemetry: make(map[int]model.Telemetry),
		byVehicle: make(map[int][]model.Telemetry),
		current:   make(map[int]model.Telemetry),
	}
}

// Save для MemoryTelemetryRepository сохраняет телеметрию в память
// Возвращает ошибку
func (r *MemoryTelemetryRepository) Save(ctx context.Context, t *model.Telemetry) error {
	r.nextID++
	t.TelemetryID = r.nextID
	r.telemetry[t.TelemetryID] = *t
	r.byVehicle[t.VehicleID] = append(r.byVehicle[t.VehicleID], *t)
	r.current[t.VehicleID] = *t
	return nil
}

// matchesFilter проверяет все непустые поля фильтра
func matchesFilter(t model.Telemetry, f model.TelemetryFilter) bool {
	if f.OrganizationID != nil && t.OrganizationID != *f.OrganizationID {
		return false
	}
	if f.VehicleID != nil && t.VehicleID != *f.VehicleID {
		return false
	}
	if f.DeviceID != nil && t.DeviceID != *f.DeviceID {
		return false
	}
	if f.FuelMin != nil && t.Fuel < *f.FuelMin {
		return false
	}
	if f.FuelMax != nil && t.Fuel > *f.FuelMax {
		return false
	}
	if f.LatMin != nil && t.Lat < *f.LatMin {
		return false
	}
	if f.LatMax != nil && t.Lat > *f.LatMax {
		return false
	}
	if f.LonMin != nil && t.Lon < *f.LonMin {
		return false
	}
	if f.LonMax != nil && t.Lon > *f.LonMax {
		return false
	}
	if f.From != nil && t.ReceivedAt.Before(*f.From) {
		return false
	}
	if f.To != nil && t.ReceivedAt.After(*f.To) {
		return false
	}

	return true
}

// GetList для MemoryTelemetryRepository возвращает полный список телеметрии
func (r *MemoryTelemetryRepository) GetList(ctx context.Context, filter model.TelemetryFilter) ([]model.Telemetry, error) {
	res := make([]model.Telemetry, 0, filter.Limit)
	skipped := 0
	for _, t := range r.telemetry {
		if !matchesFilter(t, filter) {
			continue
		}
		if skipped < filter.Offset {
			skipped++
			continue
		}
		if len(res) >= filter.Limit {
			break
		}
		res = append(res, t)
	}
	return res, nil
}

// GetItemByID для MemoryTelemetryRepository возвращает конкретную запись телеметрии по её ID
func (r *MemoryTelemetryRepository) GetItemByID(ctx context.Context, id int) (model.Telemetry, error) {
	res, ok := r.telemetry[id]
	if !ok {
		return model.Telemetry{}, model.ErrNotFound
	}
	return res, nil
}

// GetListByVehicle для MemoryTelemetryRepository возвращает всю телеметрию для конкретной машины
func (r *MemoryTelemetryRepository) GetListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	res, ok := r.byVehicle[id]
	if !ok {
		return []model.Telemetry{}, model.ErrNotFound
	}
	return res, nil
}

// DeleteListByVehicle для MemoryTelemetryRepository удаляет всю телеметрию для конкретной машины
func (r *MemoryTelemetryRepository) DeleteListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error) {
	deleted, ok := r.byVehicle[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	delete(r.byVehicle, id)
	delete(r.current, id)
	for _, t := range deleted {
		delete(r.telemetry, t.TelemetryID)
	}
	return deleted, nil
}

// DeleteItemByID для MemoryTelemetryRepository удаляет телеметрию по её ID
func (r *MemoryTelemetryRepository) DeleteItemByID(ctx context.Context, id int) (model.Telemetry, error) {
	t, ok := r.telemetry[id]
	if !ok {
		return model.Telemetry{}, model.ErrNotFound
	}
	delete(r.telemetry, id)

	list := r.byVehicle[t.VehicleID]
	for i, item := range list {
		if item.TelemetryID == id {
			r.byVehicle[t.VehicleID] = append(list[:i], list[i+1:]...)
			break
		}
	}

	if r.current[t.VehicleID].TelemetryID == id {
		delete(r.current, t.VehicleID)
	}
	return t, nil
}
