package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fleettrack/internal/model"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TelegramSender отправляет уведомления через Telegram Bot API
type TelegramSender struct {
	botToken   string
	httpClient *http.Client
}

// NewTelegramSender создаёт экземпляр отправителя в Telegram
func NewTelegramSender(botToken string, client *http.Client) *TelegramSender {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return &TelegramSender{
		botToken:   botToken,
		httpClient: client,
	}
}

// Send отправляет форматированное сообщение об алерте в чат Telegram
func (s *TelegramSender) Send(ctx context.Context, ch model.UserNotificationChannel, alert model.Alert) error {
	var cfg TelegramConfig
	if err := json.Unmarshal(ch.Config, &cfg); err != nil {
		return model.ErrInvalidChannelConfig
	}

	chatID := cfg.ChatID.String()
	if chatID == "" || chatID == "0" {
		return model.ErrInvalidChannelConfig
	}

	text := FormatTelegramMessage(alert)
	payload := struct {
		ChatID    string `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode"`
	}{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram api error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}
