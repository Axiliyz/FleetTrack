// Package notifier содержит тесты компонентов отправки уведомлений
package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"fleettrack/internal/config"
	"fleettrack/internal/model"
	"strings"
	"testing"
	"time"
)

// mockSender имитирует отправку в канал для тестов диспетчера
type mockSender struct {
	called bool
	err    error
}

func (m *mockSender) Send(ctx context.Context, ch model.UserNotificationChannel, alert model.Alert) error {
	m.called = true
	return m.err
}

// mockLogger реализует интерфейс logger.Logger для тестов
type mockLogger struct{}

func (m *mockLogger) Debug(msg string) {}
func (m *mockLogger) Info(msg string)  {}
func (m *mockLogger) Warn(msg string)  {}
func (m *mockLogger) Error(msg string) {}

// TestSeverityBadge проверяет корректность текстового бейджа для каждого уровня критичности
func TestSeverityBadge(t *testing.T) {
	tests := []struct {
		severity model.SeverityLevel
		want     string
	}{
		{model.SeverityLevelCritical, "КРИТИЧЕСКИЙ"},
		{model.SeverityLevelHigh, "ВЫСОКИЙ"},
		{model.SeverityLevelMedium, "СРЕДНИЙ"},
		{model.SeverityLevelLow, "НИЗКИЙ"},
		{model.SeverityLevel("UNKNOWN"), "UNKNOWN"},
	}

	for _, tt := range tests {
		got := severityBadge(tt.severity)
		if got != tt.want {
			t.Errorf("severityBadge(%s) = %q, want %q", tt.severity, got, tt.want)
		}
	}
}

// TestFormatTelegramMessage проверяет генерацию HTML разметки сообщения для Telegram
func TestFormatTelegramMessage(t *testing.T) {
	val := 95.5
	createdAt := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	alert := model.Alert{
		ID:             1,
		OrganizationID: 10,
		VehicleID:      42,
		Type:           model.AlertRuleSpeedExceed,
		Severity:       model.SeverityLevelHigh,
		Status:         model.AlertStatusFired,
		Message:        "Превышена скорость",
		Value:          &val,
		CreatedAt:      createdAt,
	}

	msg := FormatTelegramMessage(alert)

	if !strings.Contains(msg, "ВЫСОКИЙ АЛЕРТ #1") {
		t.Errorf("expected header with high severity badge, got: %s", msg)
	}
	if !strings.Contains(msg, "42") {
		t.Errorf("expected vehicle id 42, got: %s", msg)
	}
	if !strings.Contains(msg, "95.5") {
		t.Errorf("expected value 95.5, got: %s", msg)
	}
	if !strings.Contains(msg, "Превышена скорость") {
		t.Errorf("expected message text, got: %s", msg)
	}

	// Проверка случая когда Value == nil
	alertNoVal := model.Alert{
		ID:        2,
		Severity:  model.SeverityLevelLow,
		CreatedAt: createdAt,
	}
	msgNoVal := FormatTelegramMessage(alertNoVal)
	if !strings.Contains(msgNoVal, "—") {
		t.Errorf("expected dash for nil value, got: %s", msgNoVal)
	}
}

// TestDispatcherDispatch проверяет корректную маршрутизацию сообщений по типам каналов
func TestDispatcherDispatch(t *testing.T) {
	tgMock := &mockSender{}
	emailMock := &mockSender{}
	l := &mockLogger{}
	dispatcher := NewDispatcher(tgMock, emailMock, l)

	ctx := context.Background()
	alert := model.Alert{ID: 1}

	// Маршрутизация в Telegram
	tgChannel := model.UserNotificationChannel{Type: model.NotificationChannelTg}
	err := dispatcher.Dispatch(ctx, tgChannel, alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tgMock.called {
		t.Error("expected Telegram sender to be called")
	}

	// Маршрутизация в Email
	emailChannel := model.UserNotificationChannel{Type: model.NotificationChannelEmail}
	err = dispatcher.Dispatch(ctx, emailChannel, alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emailMock.called {
		t.Error("expected Email sender to be called")
	}

	// Неподдерживаемый тип канала
	unknownChannel := model.UserNotificationChannel{Type: "WEBHOOK"}
	err = dispatcher.Dispatch(ctx, unknownChannel, alert)
	if !errors.Is(err, model.ErrUnsupportedChannelType) {
		t.Errorf("expected ErrUnsupportedChannelType, got: %v", err)
	}
}

// TestTelegramSenderInvalidConfig проверяет валидацию некорректного конфига Telegram
func TestTelegramSenderInvalidConfig(t *testing.T) {
	sender := NewTelegramSender("fake-token", nil)
	ctx := context.Background()
	alert := model.Alert{ID: 1}

	// Невалидный JSON конфиг
	chInvalidJSON := model.UserNotificationChannel{
		Type:   model.NotificationChannelTg,
		Config: json.RawMessage(`{bad-json}`),
	}
	err := sender.Send(ctx, chInvalidJSON, alert)
	if !errors.Is(err, model.ErrInvalidChannelConfig) {
		t.Errorf("expected ErrInvalidChannelConfig for bad json, got: %v", err)
	}

	// Пустой chat_id
	chEmptyChat := model.UserNotificationChannel{
		Type:   model.NotificationChannelTg,
		Config: json.RawMessage(`{"chat_id": ""}`),
	}
	err = sender.Send(ctx, chEmptyChat, alert)
	if !errors.Is(err, model.ErrInvalidChannelConfig) {
		t.Errorf("expected ErrInvalidChannelConfig for empty chat_id, got: %v", err)
	}
}

// TestEmailSenderInvalidConfig проверяет валидацию некорректного конфига Email
func TestEmailSenderInvalidConfig(t *testing.T) {
	sender := NewEmailSender(config.SMTPConfig{})
	ctx := context.Background()
	alert := model.Alert{ID: 1}

	// Невалидный JSON
	chInvalidJSON := model.UserNotificationChannel{
		Type:   model.NotificationChannelEmail,
		Config: json.RawMessage(`{bad-json}`),
	}
	err := sender.Send(ctx, chInvalidJSON, alert)
	if !errors.Is(err, model.ErrInvalidChannelConfig) {
		t.Errorf("expected ErrInvalidChannelConfig, got: %v", err)
	}

	// Пустой Email
	chEmptyEmail := model.UserNotificationChannel{
		Type:   model.NotificationChannelEmail,
		Config: json.RawMessage(`{"email": ""}`),
	}
	err = sender.Send(ctx, chEmptyEmail, alert)
	if !errors.Is(err, model.ErrInvalidChannelConfig) {
		t.Errorf("expected ErrInvalidChannelConfig for empty email, got: %v", err)
	}
}
