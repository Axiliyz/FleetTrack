package model

import "time"

// NotificationStatus отражает состояние задачи в очереди Transactional Outbox
type NotificationStatus string

const (
	// NotificationStatusPending — задача ожидает отправки фоновым воркером
	NotificationStatusPending NotificationStatus = "PENDING"

	// NotificationStatusSent — уведомление успешно доставлено в целевой канал
	NotificationStatusSent NotificationStatus = "SENT"

	// NotificationStatusFailed — отправка не удалась после исчерпания всех попыток ретрая
	NotificationStatusFailed NotificationStatus = "FAILED"
)

// AlertNotification представляет отдельную задачу на отправку уведомления об алерте в конкретный канал
// Является частью реализации паттерна Transactional Outbox для надёжной асинхронной доставки
type AlertNotification struct {
	ID          int                // Уникальный идентификатор задачи
	AlertID     int                // Идентификатор сработавшего алерта
	ChannelID   int                // Идентификатор целевого канала доставки
	Status      NotificationStatus // Текущий статус отправки
	Attempts    int                // Количество совершённых попыток отправки
	NextRetryAt *time.Time         // Время следующей попытки при ошибке (exponential backoff)
	SentAt      *time.Time         // Время фактической успешной отправки
	Error       *string            // Текст последней ошибки при неудачной попытке
	CreatedAt   time.Time          // Время постановки задачи в очередь
}
