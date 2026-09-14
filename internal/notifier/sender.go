package notifier

import (
	"context"
	"encoding/json"
	"fleettrack/internal/model"
	"fmt"
	"time"
)

// Sender определяет интерфейс отправки уведомлений об алерте в конкретный канал
type Sender interface {
	Send(ctx context.Context, ch model.UserNotificationChannel, alert model.Alert) error
}

// TelegramConfig содержит конфигурацию канала доставки Telegram
type TelegramConfig struct {
	ChatID json.Number `json:"chat_id"`
}

// EmailConfig содержит конфигурацию канала доставки Email
type EmailConfig struct {
	Email string `json:"email"`
}

// severityBadge возвращает понятную плашку с эмодзи в зависимости от критичности
func severityBadge(s model.SeverityLevel) string {
	switch s {
	case model.SeverityLevelCritical:
		return "КРИТИЧЕСКИЙ"
	case model.SeverityLevelHigh:
		return "ВЫСОКИЙ"
	case model.SeverityLevelMedium:
		return "СРЕДНИЙ"
	case model.SeverityLevelLow:
		return "НИЗКИЙ"
	default:
		return string(s)
	}
}

// FormatTelegramMessage формирует отформатированное HTML-сообщение для отправки в Telegram
func FormatTelegramMessage(alert model.Alert) string {
	valStr := "—"
	if alert.Value != nil {
		valStr = fmt.Sprintf("%.1f", *alert.Value)
	}

	createdAtStr := alert.CreatedAt.Format("02.01.2006 15:04:05")
	if alert.CreatedAt.IsZero() {
		createdAtStr = time.Now().Format("02.01.2006 15:04:05")
	}

	return fmt.Sprintf(
		"<b>%s АЛЕРТ #%d</b>\n\n"+
			"🚗 <b>Автомобиль ID:</b> <code>%d</code>\n"+
			"🏢 <b>Организация ID:</b> <code>%d</code>\n"+
			"⚡ <b>Тип:</b> <code>%s</code>\n"+
			"📊 <b>Значение:</b> <code>%s</code>\n"+
			"📝 <b>Сообщение:</b> %s\n"+
			"🕒 <b>Время:</b> %s",
		severityBadge(alert.Severity),
		alert.ID,
		alert.VehicleID,
		alert.OrganizationID,
		alert.Type,
		valStr,
		alert.Message,
		createdAtStr,
	)
}
