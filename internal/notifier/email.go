package notifier

import (
	"context"
	"encoding/json"
	"fleettrack/internal/config"
	"fleettrack/internal/model"
	"fmt"
	"net/smtp"
)

// EmailSender отправляет уведомления по электронной почте через SMTP
type EmailSender struct {
	cfg config.SMTPConfig
}

// NewEmailSender создаёт экземпляр отправителя Email
func NewEmailSender(cfg config.SMTPConfig) *EmailSender {
	return &EmailSender{
		cfg: cfg,
	}
}

// Send отправляет письмо об инциденте на указанный адрес электронной почты
func (s *EmailSender) Send(ctx context.Context, ch model.UserNotificationChannel, alert model.Alert) error {
	var cfg EmailConfig
	if err := json.Unmarshal(ch.Config, &cfg); err != nil {
		return model.ErrInvalidChannelConfig
	}

	if cfg.Email == "" {
		return model.ErrInvalidChannelConfig
	}

	from := s.cfg.From
	if from == "" {
		from = s.cfg.Username
	}
	to := cfg.Email
	subject := fmt.Sprintf("[FleetTrack] Алерт %s: Автомобиль #%d (%s)", alert.Severity, alert.VehicleID, alert.Type)

	body := fmt.Sprintf("Зафиксирован инцидент:\n\nАвтомобиль: %d\nКритичность: %s\nТип: %s\nСообщение: %s\nВремя: %s",
		alert.VehicleID, alert.Severity, alert.Type, alert.Message, alert.CreatedAt.Format("02.01.2006 15:04:05"))
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body))

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
