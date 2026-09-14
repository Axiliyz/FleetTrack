package notifier

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
)

// Dispatcher маршрутизирует отправку уведомлений в зависимости от типа канала
type Dispatcher struct {
	TelegramSender Sender
	EmailSender    Sender
	logger         logger.Logger
}

// NewDispatcher создаёт диспетчер отправки уведомлений
func NewDispatcher(tg Sender, email Sender, l logger.Logger) *Dispatcher {
	return &Dispatcher{
		TelegramSender: tg,
		EmailSender:    email,
		logger:         l,
	}
}

// Dispatch отправляет уведомление в соответствующий канал доставки
func (d *Dispatcher) Dispatch(ctx context.Context, ch model.UserNotificationChannel, alert model.Alert) error {
	switch ch.Type {
	case model.NotificationChannelTg:
		return d.TelegramSender.Send(ctx, ch, alert)
	case model.NotificationChannelEmail:
		return d.EmailSender.Send(ctx, ch, alert)
	default:
		return model.ErrUnsupportedChannelType
	}
}
