// Package repository определяет контракты доступа к данным для всех доменных сущностей.
package repository

import (
	"context"
	"fleettrack/internal/model"
)

// AssignmentRepository описывает доступ к связям устройств и автомобилей
type AssignmentRepository interface {
	// GetActiveAssignment возвращает активную (незавершённую) связь устройства с автомобилем
	// Возвращает ошибку, если активной связи не найдено
	GetActiveAssignment(ctx context.Context, deviceID int) (model.DeviceAssignment, error)
	// CreateAssignment создаёт новую связь устройства с автомобилем
	// Возвращает ошибку, если не удалось сохранить
	CreateAssignment(ctx context.Context, assignment *model.DeviceAssignment) error
	// EndAssignment завершает активную связь устройства с автомобилем
	// Возвращает ошибку, если завершить связь не удалось
	EndAssignment(ctx context.Context, deviceID int) error
}

// TelemetryRepository определяет контракт сохранения телеметрии
type TelemetryRepository interface {
	// Save сохраняет телеметрию в хранилище
	// Возвращает ошибку если сохранение не удалось
	Save(ctx context.Context, t *model.Telemetry) error
	// GetList возвращает список всей телеметрии
	GetList(ctx context.Context, filter model.TelemetryFilter) ([]model.Telemetry, error)
	// GetItemByID возвращает запись телеметрии по её ID
	GetItemByID(ctx context.Context, id int) (model.Telemetry, error)
	// GetListByVehicle возвращает срез телеметрий по ID машины
	GetListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error)
	// DeleteItemByID удаляет запись по её ID
	DeleteItemByID(ctx context.Context, id int) (model.Telemetry, error)
	// DeleteListByVehicle удаляет список записей по ID машины
	DeleteListByVehicle(ctx context.Context, id int) ([]model.Telemetry, error)
}

// VehicleRepository определяет контракт хранения автомобилей в системе
type VehicleRepository interface {
	// Create создаёт новый автомобиль
	// Возвращает ошибку если не удалось
	Create(ctx context.Context, v *model.Vehicle) error
	// GetByID возвращает данные по автомобилю по его ID
	// Или ошибку, если не нашлось
	GetByID(ctx context.Context, id int) (model.Vehicle, error)
	// GetList возвращает срез автомобилей
	// Или ошибку
	GetList(ctx context.Context, filter model.VehicleFilter) ([]model.Vehicle, error)
	// Delete удаляет машину по ID
	// Возвращает удалённую машину, и ошибку, если не удалось
	Delete(ctx context.Context, id int) (model.Vehicle, error)
	// Update обновляет некоторые поля по авто
	Update(ctx context.Context, id int, upd model.UpdateVehicle) (model.Vehicle, error)
}

// DeviceRepository описывает доступ к устройствам, необходимый сервису связей.
type DeviceRepository interface {
	// GetByID возвращает данные по девайсу по его ID
	// Или ошибку, если не нашлось
	GetByID(ctx context.Context, deviceID int) (model.Device, error)
	// Create создаёт новый девайс
	// Возвращает ошибку если не удалось
	Create(ctx context.Context, d *model.Device) error
	// Delete удаляет девайс по ID
	// Возвращает удалённую запись или ошибку
	Delete(ctx context.Context, id int) (model.Device, error)
}

// OrgRepository определяет контракт хранения организаций
type OrgRepository interface {
	// CreateOrg создаёт новую организацию
	// Возвращает ошибку если не удалось
	CreateOrg(ctx context.Context, o *model.Org) error
}

// DriverRepository определяет контракт хранения водителей
type DriverRepository interface {
	// Create создаёт нового водителя
	// Возвращает ошибку если не удалось
	Create(ctx context.Context, d *model.Driver) error
	// GetByID возвращает водителя по его ID
	// Или ошибку, если не нашлось
	GetByID(ctx context.Context, id int) (model.Driver, error)
	// GetList возвращает срез водителей
	// Или ошибку
	GetList(ctx context.Context, filter model.DriverFilter) ([]model.Driver, error)
	// Delete удаляет водителя по ID
	// Возвращает удалённую запись, и ошибку, если не удалось
	Delete(ctx context.Context, id int) (model.Driver, error)
	// Update обновляет некоторые поля водителя
	Update(ctx context.Context, id int, upd model.UpdateDriver) (model.Driver, error)
}

// TripRepository задаёт контракт хранения рейсов
type TripRepository interface {
	// CreateTrip создаёт новую поездку, заполняя ID, StartedAt и Status в t
	// Возвращает ошибку, если не удалось
	CreateTrip(ctx context.Context, t *model.Trip) error

	// GetListTrips возвращает список рейсов(с фильтрами в Query параметрах)
	// Или ошибку, если не нашлось
	GetListTrips(ctx context.Context, f *model.TripFilter) ([]model.Trip, error)

	// UpdateTrip обновляет статус рейса
	// Возвращает новый объект рейса, либо ошибку
	UpdateTrip(ctx context.Context, upd model.Trip) (model.Trip, error)

	// DeleteTrip выставляет статус Cancelled по ID и выставляет время
	// Возвращает удалённую запись или ошибку
	DeleteTrip(ctx context.Context, id int) (model.Trip, error)
}
