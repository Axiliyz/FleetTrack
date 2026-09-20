package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fleettrack/internal/logger"
	"fleettrack/internal/repository"
	"fleettrack/internal/service"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TelegramWorker опрашивает Telegram Bot API методом long polling и обрабатывает callback-запросы от inline-кнопок
type TelegramWorker struct {
	botToken     string
	alertService *service.AlertService
	channelRepo  repository.NotificationChannelRepository
	httpClient   *http.Client
	logger       logger.Logger
	offset       int
	wg           sync.WaitGroup
}

// NewTelegramWorker создаёт экземпляр фонового воркера для обработки callback-запросов Telegram
func NewTelegramWorker(
	token string,
	as *service.AlertService,
	channelRepo repository.NotificationChannelRepository,
	client *http.Client,
	l logger.Logger,
) *TelegramWorker {
	if client == nil {
		client = &http.Client{
			Timeout: 35 * time.Second,
		}
	}
	return &TelegramWorker{
		botToken:     token,
		alertService: as,
		channelRepo:  channelRepo,
		httpClient:   client,
		logger:       l,
		offset:       0,
	}
}

type getUpdatesResponse struct {
	OK     bool             `json:"ok"`
	Result []telegramUpdate `json:"result"`
}

type telegramUpdate struct {
	UpdateID      int                    `json:"update_id"`
	CallbackQuery *telegramCallbackQuery `json:"callback_query"`
}

type telegramCallbackQuery struct {
	ID      string               `json:"id"`
	From    telegramUser         `json:"from"`
	Message *telegramMessageInfo `json:"message"`
	Data    string               `json:"data"`
}
type telegramUser struct {
	ID int64 `json:"id"`
}
type telegramMessageInfo struct {
	MessageID int              `json:"message_id"`
	Chat      telegramChatInfo `json:"chat"`
}
type telegramChatInfo struct {
	ID int64 `json:"id"`
}

// Start запускает цикл long polling в отдельной горутине
func (w *TelegramWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
}

// Stop ожидает корректного завершения фоновой горутины
func (w *TelegramWorker) Stop() {
	w.wg.Wait()
}

// run циклически вызывает poll до отмены контекста
func (w *TelegramWorker) run(ctx context.Context) {
	defer w.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := w.poll(ctx); err != nil {
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
				}
			}
		}
	}
}

// poll выполняет один запрос getUpdates к Telegram Bot API с заданным таймаутом
func (w *TelegramWorker) poll(ctx context.Context) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=25", w.botToken, w.offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		w.logger.Warn(fmt.Sprintf("failed to create telegram poll request: %s", err.Error()))
		return err
	}
	resp, err := w.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		w.logger.Error(fmt.Sprintf("failed to poll telegram updates: %s", err.Error()))
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram getUpdates error (status %d): %s", resp.StatusCode, string(body))
	}

	var data getUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	for _, u := range data.Result {
		if u.UpdateID >= w.offset {
			w.offset = u.UpdateID + 1
		}
		if u.CallbackQuery != nil {
			w.handleCallbackQuery(ctx, u.CallbackQuery)
		}
	}
	return nil
}

// handleCallbackQuery обрабатывает входящий callback-запрос взятия алерта в работу
func (w *TelegramWorker) handleCallbackQuery(ctx context.Context, cq *telegramCallbackQuery) {
	if !strings.HasPrefix(cq.Data, "ack:") {
		return
	}

	alertIDStr := strings.TrimPrefix(cq.Data, "ack:")
	alertID, err := strconv.Atoi(alertIDStr)
	if err != nil {
		w.logger.Warn(fmt.Sprintf("invalid alert id in callback: %s", cq.Data))
		return
	}

	var chatID int64
	if cq.Message != nil {
		chatID = cq.Message.Chat.ID
	} else {
		chatID = cq.From.ID
	}

	userID, err := w.channelRepo.GetUserIDByTelegramChatID(ctx, int(chatID))
	if err != nil {
		w.logger.Warn(fmt.Sprintf("unknown telegram user/chat %d: %s", chatID, err.Error()))
		w.answerCallbackQuery(ctx, cq.ID, "Telegram-чат не привязан к пользователю FleetTrack")
		return
	}

	_, err = w.alertService.AcknowledgeAlert(ctx, alertID, userID)
	if err != nil {
		w.logger.Warn(fmt.Sprintf("failed to ack alert %d: %s", alertID, err.Error()))
		w.answerCallbackQuery(ctx, cq.ID, "не удалось принять алерт")
		return
	}

	w.answerCallbackQuery(ctx, cq.ID, fmt.Sprintf("Алерт #%d взят в работу!", alertID))

	if cq.Message != nil {
		w.editMessageButton(ctx, cq.Message.Chat.ID, cq.Message.MessageID, "В работе")
	}

	w.logger.Info(fmt.Sprintf("telegram callback handled: alert %d acknowledged by user %d", alertID, userID))
}

// answerCallbackQuery отправляет подтверждение Telegram о получении клика
func (w *TelegramWorker) answerCallbackQuery(ctx context.Context, callbackID, text string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", w.botToken)
	payload := map[string]string{
		"callback_query_id": callbackID,
		"text":              text,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		w.logger.Warn(fmt.Sprintf("failed to create answerCallbackQuery request: %s", err.Error()))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		w.logger.Warn(fmt.Sprintf("failed to execute answerCallbackQuery: %s", err.Error()))
		return
	}
	_ = resp.Body.Close()
}

// editMessageButton обновляет кнопку под сообщением на статусную
func (w *TelegramWorker) editMessageButton(ctx context.Context, chatID int64, messageID int, text string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageReplyMarkup", w.botToken)
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"reply_markup": map[string]any{
			"inline_keyboard": [][]map[string]string{
				{
					{
						"text":          text,
						"callback_data": "noop",
					},
				},
			},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		w.logger.Warn(fmt.Sprintf("failed to create editMessageReplyMarkup request: %s", err.Error()))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		w.logger.Warn(fmt.Sprintf("failed to execute editMessageReplyMarkup: %s", err.Error()))
		return
	}
	_ = resp.Body.Close()
}
