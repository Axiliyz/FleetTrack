// Package memory содержит in-memory реализации репозиториев
package memory

import (
	"context"
	"fleettrack/internal/model"
	"time"
)

// MemoryVehicleRepository хранит автомобили в памяти процесса
type MemoryVehicleRepository struct {
	vehicles map[int]model.Vehicle
	nextID   int
}

// NewMemoryVehicleRepository создаёт новый in-memory репозиторий автомобилей
func NewMemoryVehicleRepository() *MemoryVehicleRepository {
	return &MemoryVehicleRepository{
		vehicles: make(map[int]model.Vehicle),
	}
}

// Create для MemoryVehicleRepository сохраняет новую машину в памяти
func (r *MemoryVehicleRepository) Create(ctx context.Context, v *model.Vehicle) error {
	r.nextID++
	v.ID = r.nextID
	v.CreatedAt = time.Now()
	r.vehicles[v.ID] = *v
	return nil
}

// matchesVehicleFilter проверяет все непустые поля фильтра
func matchesVehicleFilter(v model.Vehicle, f model.VehicleFilter) bool {
	if f.OrganizationID != nil && v.OrganizationID != *f.OrganizationID {
		return false
	}
	if f.Model != nil && v.Model != *f.Model {
		return false
	}
	if f.VIN != nil && v.VIN != *f.VIN {
		return false
	}
	if f.Status != nil && v.Status != *f.Status {
		return false
	}
	if f.NumberPlate != nil && v.NumberPlate != *f.NumberPlate {
		return false
	}
	if f.CreatedTo != nil && v.CreatedAt.After(*f.CreatedTo) {
		return false
	}
	if f.CreatedFrom != nil && v.CreatedAt.Before(*f.CreatedFrom) {
		return false
	}
	return true
}

// GetList для MemoryVehicleRepository возвращает полный список автомобилей
func (r *MemoryVehicleRepository) GetList(ctx context.Context, filter model.VehicleFilter) ([]model.Vehicle, error) {
	list := make([]model.Vehicle, 0, filter.Limit)
	skipped := 0
	for _, v := range r.vehicles {
		if !matchesVehicleFilter(v, filter) {
			continue
		}
		if skipped < filter.Offset {
			skipped++
			continue
		}
		if len(list) >= filter.Limit {
			break
		}
		list = append(list, v)
	}
	return list, nil
}

// GetByID для MemoryVehicleRepository возвращает конкретный авто по ID
func (r *MemoryVehicleRepository) GetByID(ctx context.Context, id int) (model.Vehicle, error) {
	res, ok := r.vehicles[id]
	if !ok {
		return model.Vehicle{}, model.ErrNotFound
	}
	return res, nil
}

// Delete для MemoryVehicleRepository удаляет авто по ID (soft delete)
func (r *MemoryVehicleRepository) Delete(ctx context.Context, id int) (model.Vehicle, error) {
	v, ok := r.vehicles[id]
	if !ok {
		return model.Vehicle{}, model.ErrNotFound
	}
	v.Status = model.VehicleStatusDeleted
	r.vehicles[id] = v
	return v, nil
}

// Update для MemoryVehicleRepository обновляет некоторые данные авто по ID
func (r *MemoryVehicleRepository) Update(ctx context.Context, id int, upd model.UpdateVehicle) (model.Vehicle, error) {
	v, ok := r.vehicles[id]
	if !ok {
		return model.Vehicle{}, model.ErrNotFound
	}
	if upd.OrganizationID != nil {
		v.OrganizationID = *upd.OrganizationID
	}
	if upd.NumberPlate != nil {
		v.NumberPlate = *upd.NumberPlate
	}
	if upd.Status != nil {
		v.Status = *upd.Status
	}
	now := time.Now()
	v.UpdatedAt = &now
	r.vehicles[id] = v
	return v, nil
}
