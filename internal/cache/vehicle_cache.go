package cache

import (
	"fleettrack/internal/model"
	"sync"
)

// VehicleCache — потокобезопасный in-memory кэш автомобилей
// Кэш предназначен для быстрого доступа к данным без обращения к базе данных
//   - Get и Count используют RLock, позволяя нескольким горутинам одновременно читать данные
//   - Set, Delete и Clear используют Lock, обеспечивая эксклюзивный доступ при изменении кэша
//
// VehicleCache не является источником истины
// Источником истины остается база данных, а кэш хранит лишь копию актуального состояния
type VehicleCache struct {
	mu       sync.RWMutex
	vehicles map[int]model.Vehicle
}

// NewVehicleCache создает и возвращает пустой экземпляр VehicleCache
func NewVehicleCache() *VehicleCache {
	return &VehicleCache{
		vehicles: make(map[int]model.Vehicle),
	}
}

// Get возвращает автомобиль по его ID
// Второе возвращаемое значение показывает, найден ли автомобиль в кэше
func (vc *VehicleCache) Get(id int) (model.Vehicle, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	res, ok := vc.vehicles[id]
	return res, ok
}

// Set добавляет новый автомобиль в кэш или обновляет существующий по его ID
func (vc *VehicleCache) Set(vehicle model.Vehicle) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.vehicles[vehicle.ID] = vehicle
}

// Delete удаляет автомобиль из кэша по ID
// Возвращает удаленный автомобиль и признак успешного удаления
func (vc *VehicleCache) Delete(id int) (model.Vehicle, bool) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	deleted, ok := vc.vehicles[id]
	delete(vc.vehicles, id)

	return deleted, ok
}

// Count возвращает текущее количество автомобилей в кэше
func (vc *VehicleCache) Count() int {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	return len(vc.vehicles)
}

// Clear удаляет все автомобили из кэша
func (vc *VehicleCache) Clear() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	clear(vc.vehicles)
}
