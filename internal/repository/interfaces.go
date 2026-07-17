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
	Delete(ctx context.Context, id int) (model.Device, error)
}
