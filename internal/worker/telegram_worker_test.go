// Package worker содержит тесты фоновых воркеров приложения
package worker

import (
	"context"
	"testing"
	"time"
)

// TestNewTelegramWorker_Defaults проверяет инициализацию воркера со значениями по умолчанию
func TestNewTelegramWorker_Defaults(t *testing.T) {
	logger := &mockLogger{}
	w := NewTelegramWorker("test-token", nil, nil, nil, logger)

	if w.botToken != "test-token" {
		t.Errorf("expected botToken test-token, got %s", w.botToken)
	}
	if w.offset != 0 {
		t.Errorf("expected initial offset 0, got %d", w.offset)
	}
	if w.httpClient == nil || w.httpClient.Timeout < 30*time.Second {
		t.Errorf("expected httpClient with timeout >= 30s, got %v", w.httpClient)
	}
}

// TestTelegramWorker_HandleCallbackQuery_InvalidData проверяет игнорирование неизвестных данных callback
func TestTelegramWorker_HandleCallbackQuery_InvalidData(t *testing.T) {
	logger := &mockLogger{}
	w := NewTelegramWorker("test-token", nil, nil, nil, logger)

	ctx := context.Background()

	// Неизвестный префикс
	cqUnknown := &telegramCallbackQuery{
		ID:   "cb1",
		Data: "unknown:123",
	}
	w.handleCallbackQuery(ctx, cqUnknown)

	// Невалидный ID
	cqInvalidID := &telegramCallbackQuery{
		ID:   "cb2",
		Data: "ack:abc",
	}
	w.handleCallbackQuery(ctx, cqInvalidID)
}
