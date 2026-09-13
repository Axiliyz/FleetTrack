package model

import (
	"encoding/json"
	"time"
)

// NotificationChannelType определяет поддерживаемый транспорт отправки уведомлений
type NotificationChannelType string

const (
	// NotificationChannelEmail — доставка уведомления через электронную почту (SMTP)
	NotificationChannelEmail NotificationChannelType = "EMAIL"

	// NotificationChannelTg — доставка уведомления в Telegram-чат или бот
	NotificationChannelTg NotificationChannelType = "TELEGRAM"

	// NotificationChannelWeb — доставка уведомления по HTTP POST вебхуку на внешний URL
	NotificationChannelWeb NotificationChannelType = "WEBHOOK"
)

// UserNotificationChannel описывает персональный канал доставки уведомлений пользователя
// Хранит настройки авторизации/адресации в виде JSONB-конфигурации
type UserNotificationChannel struct {
	ID          int                     // Уникальный идентификатор канала
	UserID      int                     // Идентификатор пользователя-владельца
	Type        NotificationChannelType // Тип транспорта (Email, Telegram, Webhook)
	Config      json.RawMessage         // Специфичная для типа конфигурация (chat_id, email адрес, webhook url)
	Enabled     bool                    // Флаг включения доставки в данный канал
	MinSeverity SeverityLevel           // Минимальная критичность алертов для отправки в этот канал
	CreatedAt   time.Time               // Время добавления канала
}
