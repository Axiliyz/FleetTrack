// Package repository определяет контракты доступа к данным для всех доменных сущностей.
package repository

import (
	"context"
	"fleettrack/internal/model"
	"time"
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
	// Возвращает ошибку, если не может найти
	GetList(ctx context.Context, filter model.TelemetryFilter) ([]model.Telemetry, error)

	// GetItemByID возвращает запись телеметрии по её ID
	// Возвращает ошибку, если не может найти
	GetItemByID(ctx context.Context, id int) (model.Telemetry, error)

	// GetListByVehicle возвращает срез телеметрий по ID машины
	// Возвращает ошибку, если не может найти
	GetListByVehicle(ctx context.Context, id int, organizationID *int) ([]model.Telemetry, error)

	// DeleteItemByID удаляет запись по её ID с ограничением по организации
	// Возвращает ошибку, если не удалось удалить
	DeleteItemByID(ctx context.Context, id int, organizationID *int) (model.Telemetry, error)

	// DeleteListByVehicle удаляет список записей по ID машины с ограничением по организации
	// Возвращает ошибку, если не удалось удалить
	DeleteListByVehicle(ctx context.Context, id int, organizationID *int) ([]model.Telemetry, error)

	// GetLastByVehicle получает последнюю телеметрию по ID машины
	// Возвращает model.ErrNotFound, если у машины ещё не было телеметрии - это
	// ожидаемый случай (первая точка), а не сбой; любая другая ошибка - реальный сбой
	GetLastByVehicle(ctx context.Context, id int) (model.Telemetry, error)
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

	// Delete удаляет машину по ID. organizationID != nil ограничивает удаление машинами
	// этой организации; nil - без ограничения (для ADMIN).
	// Возвращает удалённую машину, и ошибку, если не удалось
	Delete(ctx context.Context, id int, organizationID *int) (model.Vehicle, error)

	// Update обновляет некоторые поля по авто. organizationID != nil ограничивает обновление
	// машинами этой организации; nil - без ограничения (для ADMIN).
	Update(ctx context.Context, id int, upd model.UpdateVehicle, organizationID *int) (model.Vehicle, error)

	// UpdateLastTelemetryAt устанавливает поле last_telemetry_at, нужно для партиций
	UpdateLastTelemetryAt(ctx context.Context, vehicleID int, at time.Time) error
}

// DeviceRepository описывает доступ к устройствам, необходимый сервису связей.
type DeviceRepository interface {
	// GetByID возвращает данные по девайсу по его ID
	// Или ошибку, если не нашлось
	GetByID(ctx context.Context, deviceID int) (model.Device, error)

	// Create создаёт новый девайс
	// Возвращает ошибку если не удалось
	Create(ctx context.Context, d *model.Device) error

	// Delete удаляет девайс по ID. organizationID != nil ограничивает удаление устройствами
	// этой организации; nil - без ограничения (для ADMIN).
	// Возвращает удалённую запись или ошибку
	Delete(ctx context.Context, id int, organizationID *int) (model.Device, error)
}

// OrgRepository определяет контракт хранения организаций
type OrgRepository interface {
	// CreateOrg создаёт новую организацию
	// Возвращает ошибку если не удалось
	CreateOrg(ctx context.Context, o *model.Org) error

	// GetList возвращает организацию с данным ID в виде списка из одного элемента
	// Возвращает ошибку, если не удалось получить
	GetList(ctx context.Context, organizationID int) ([]model.Org, error)
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

	// Delete удаляет водителя по ID. organizationID != nil ограничивает удаление водителями
	// этой организации; nil - без ограничения (для ADMIN).
	// Возвращает удалённую запись, и ошибку, если не удалось
	Delete(ctx context.Context, id int, organizationID *int) (model.Driver, error)

	// Update обновляет некоторые поля водителя. organizationID != nil ограничивает обновление
	// водителями этой организации; nil - без ограничения (для ADMIN).
	Update(ctx context.Context, id int, upd model.UpdateDriver, organizationID *int) (model.Driver, error)
}

// TripRepository задаёт контракт хранения рейсов
type TripRepository interface {
	// CreateTrip создаёт новую поездку, заполняя ID, StartedAt и Status в t
	// CreateTrip создаёт рейс с проверкой принадлежности водителя и авто организации
	// Возвращает ошибку, если не удалось
	CreateTrip(ctx context.Context, t *model.Trip, organizationID *int) error

	// GetListTrips возвращает список рейсов (с фильтрами в Query параметрах)
	// Или ошибку, если не нашлось
	GetListTrips(ctx context.Context, f *model.TripFilter) ([]model.Trip, error)

	// UpdateTrip обновляет статус рейса с проверкой принадлежности к организации
	// Возвращает новый объект рейса, либо ошибку
	UpdateTrip(ctx context.Context, upd model.Trip, organizationID *int) (model.Trip, error)

	// DeleteTrip выставляет статус Cancelled по ID с проверкой принадлежности к организации
	// Возвращает удалённую запись или ошибку
	DeleteTrip(ctx context.Context, id int, organizationID *int) (model.Trip, error)

	// UpdateTripStats позволяет обновить расстояние и статистику скорости по рейсу
	// Возвращает итоговый рейс или ошибку
	UpdateTripStats(ctx context.Context, id int, distance, speed float64) (model.Trip, error)

	// GetByID возвращает рейс по его ID с проверкой принадлежности к организации
	// Возвращает model.ErrNotFound если не нашёл
	GetByID(ctx context.Context, id int, organizationID *int) (model.Trip, error)
}

// UserRepository задаёт контракт хранения пользователей
type UserRepository interface {
	// GetByEmail получает юзера по почте
	// Возвращает model.ErrNotFound если не нашёл
	GetByEmail(ctx context.Context, email string) (model.User, error)

	// GetByID получает юзера по ID
	// Возвращает найденного юзера или ошибку
	GetByID(ctx context.Context, id int) (model.User, error)

	// Create создаёт нового юзера
	// Возвращает объект юзера или ошибку
	Create(ctx context.Context, u *model.User) error

	// DeleteByID удаляет юзера по ID. organizationID != nil ограничивает удаление юзерами
	// этой организации; nil - без ограничения.
	// Возвращает удалённого юзера или ошибку
	DeleteByID(ctx context.Context, id int, organizationID *int) (model.User, error)

	// GetList получает список юзеров с фильтрами
	// Возвращает слайс юзеров или ошибку
	GetList(ctx context.Context, filter model.UserFilter) ([]model.User, error)
}

// RefreshTokenRepository задаёт контракт хранения refresh-токенов
type RefreshTokenRepository interface {
	// Create сохраняет новый refresh-токен, заполняя ID и CreatedAt
	// Возвращает ошибку если не удалось
	Create(ctx context.Context, t *model.RefreshToken) error

	// GetActiveByHash возвращает токен по его хешу, если он не отозван и не истёк
	// Возвращает model.ErrNotFound, если такого валидного токена нет
	GetActiveByHash(ctx context.Context, hash string) (model.RefreshToken, error)

	// Revoke помечает токен отозванным по его ID
	// Возвращает ошибку если не удалось
	Revoke(ctx context.Context, id int) error

	// RevokeActiveByHash помечает токен отозванным по хэшу
	// Предотвращает гонки при обновлении
	RevokeActiveByHash(ctx context.Context, hash string) (model.RefreshToken, error)
}

// AlertRuleRepository определяет контракт хранения правил для алертов
type AlertRuleRepository interface {
	// Create создаёт новое правило генерации алертов для организации
	Create(ctx context.Context, r *model.AlertRule) error

	// GetByID возвращает правило по его идентификатору
	// Возвращает model.ErrNotFound, если правило не найдено
	GetByID(ctx context.Context, id int) (model.AlertRule, error)

	// GetList возвращает список правил с фильтрацией и пагинацией
	GetList(ctx context.Context, filter model.AlertRuleFilter) ([]model.AlertRule, error)

	// Update обновляет конфигурацию существующего правила (порог, статус enabled, severity)
	Update(ctx context.Context, upd model.AlertRule) (model.AlertRule, error)

	// GetActiveRulesByOrg возвращает все включенные (enabled=true) правила организации
	// Используется Alert Engine для проверки входящей телеметрии
	GetActiveRulesByOrg(ctx context.Context, orgID int) ([]model.AlertRule, error)

	// DeleteByID удаляет правило по его ID
	DeleteByID(ctx context.Context, id int) (model.AlertRule, error)
}

// AlertRepository определяет контракт хранения и управления жизненным циклом алертов (stateful alerting)
type AlertRepository interface {
	// GetActiveAlert ищет текущий незакрытый алерт (status != RESOLVED) для конкретной машины и правила
	// Возвращает model.ErrNotFound, если активного алерта нет (использует partial unique index)
	GetActiveAlert(ctx context.Context, vehicleID, ruleID int) (model.Alert, error)

	// Create сохраняет новый сработавший алерт со статусом FIRED
	Create(ctx context.Context, a *model.Alert) error

	// Resolve переводит алерт в статус RESOLVED и проставляет время закрытия
	// resolvedBy передаётся как nil при автоматическом закрытии системой,
	// либо указывает на userID при ручном закрытии оператором
	Resolve(ctx context.Context, id int, resolvedBy *int) (model.Alert, error)

	// AcknowledgeAlert переводит алерт в статус ACKNOWLEDGED, фиксируя время и ID оператора
	AcknowledgeAlert(ctx context.Context, alertID, userID int) (model.Alert, error)

	// GetList возвращает список алертов по фильтрам (организация, статус, даты, пагинация)
	GetList(ctx context.Context, filter model.AlertFilter) ([]model.Alert, error)

	// FindOfflineVehicles ищет автомобили с активными трекерами, не присылавшие телеметрию дольше thresholdMinutes
	FindOfflineVehicles(ctx context.Context, thresholdMinutes float64) ([]model.OfflineVehicleInfo, error)

	// AcquireLock блокирует критическую секцию, пока транзакция не закрыта
	AcquireLock(ctx context.Context, vehicleID, ruleID int) error
}

// NotificationChannelRepository определяет контракт управления каналами доставки уведомлений пользователей
type NotificationChannelRepository interface {
	// Create привязывает новый канал доставки (Email, Telegram, Webhook) к пользователю
	Create(ctx context.Context, ch *model.UserNotificationChannel) error

	// GetByUserID возвращает все настроенные каналы указанного пользователя
	GetByUserID(ctx context.Context, userID int) ([]model.UserNotificationChannel, error)

	// GetChannelsForAlert выбирает все активные каналы пользователей организации,
	// у которых минимальный порог min_severity совпадает или ниже критичности алерта
	GetChannelsForAlert(ctx context.Context, orgID int, severity model.SeverityLevel) ([]model.UserNotificationChannel, error)

	// Delete удаляет канал уведомлений по ID
	Delete(ctx context.Context, id int) error

	// GetUserIDByTelegramChatID находит ID пользователя по ID чата
	GetUserIDByTelegramChatID(ctx context.Context, chatID int) (int, error)
}

// AlertNotificationRepository реализует transactional outbox паттерн для надёжной отправки уведомлений
type AlertNotificationRepository interface {
	// CreateBatch сохраняет пачку задач на отправку уведомлений со статусом PENDING
	// Должен вызываться в одной транзакции с генерацией алерта
	CreateBatch(ctx context.Context, nots []model.AlertNotification) error

	// FetchPending забирает пачку задач со статусом PENDING для отправки воркером
	// В хайлоад реализации использует FOR UPDATE SKIP LOCKED для безопасного параллелизма
	FetchPending(ctx context.Context, batchSize int) ([]model.NotificationTask, error)

	// MarkSent фиксирует успешную отправку уведомления в канал (status = SENT)
	MarkSent(ctx context.Context, id int) error

	// MarkFailed фиксирует неудачную попытку отправки (увеличивает attempts, сохраняет ошибку и выставляет next_retry_at)
	// Если исчерпан лимит попыток, переводит статус в FAILED
	MarkFailed(ctx context.Context, id int, errMsg string, nextRetry *time.Time) error
}

// PartitionRepository отвечает за создание партиций таблиц в БД
type PartitionRepository interface {
	// CreatePartition создаёт партицию таблицы на указанный диапазон дат, если она ещё не существует
	CreatePartition(ctx context.Context, parentTable, partitionName string, from, to time.Time) error
}
