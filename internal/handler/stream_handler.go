package handler

import (
	"encoding/json"
	"errors"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"fleettrack/internal/stream"
	"fmt"
	"io"
	"net/http"
	"time"
)

// streamHeartbeat — как часто слать пустой комментарий, чтобы прокси и балансировщики
// не закрыли «молчащее» соединение и браузер быстро заметил обрыв
const streamHeartbeat = 15 * time.Second

// streamRetryMs — через сколько браузеру переподключаться после обрыва (поле retry в SSE)
const streamRetryMs = 3000

// StreamSubscriber — то, что нужно хендлеру от stream.Hub
type StreamSubscriber interface {
	Subscribe(orgID int) (<-chan stream.Event, func())
}

// StreamHandler отдаёт события парка в реальном времени по Server-Sent Events
type StreamHandler struct {
	hub       StreamSubscriber
	logger    logger.Logger
	heartbeat time.Duration
}

// NewStreamHandler создаёт хендлер потока событий
func NewStreamHandler(hub StreamSubscriber, l logger.Logger) *StreamHandler {
	return &StreamHandler{hub: hub, logger: l, heartbeat: streamHeartbeat}
}

// HandleStream держит открытым поток событий организации текущего пользователя
// @Summary Поток событий парка (SSE)
// @Description Соединение не закрывается: сервер пишет события в формате text/event-stream.
// @Description Типы событий: telemetry, alert, trip, vehicle; data — тот же JSON, что в REST-ответах.
// @Description Если клиент не успевает читать, сервер закрывает поток — клиент переподключается и заново берёт /fleet/snapshot.
// @Tags fleet
// @Produce text/event-stream
// @Success 200 {string} string "поток событий"
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Router /stream [get]
// @Security BearerAuth
func (h *StreamHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		respondError(w, r, h.logger, model.ErrMissingToken)
		return
	}

	// сервер ставит общие ReadTimeout/WriteTimeout на соединение; для бесконечного потока их снимаем.
	// ReadTimeout тоже важен: по его истечении net/http отменяет контекст запроса
	rc := http.NewResponseController(w)
	for _, clear := range []func(time.Time) error{rc.SetWriteDeadline, rc.SetReadDeadline} {
		if err := clear(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
			h.logger.Warn(fmt.Sprintf("stream: can't clear connection deadline: %s", err))
		}
	}

	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	// nginx по умолчанию буферизует ответы — для потока это задержка в секунды
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	if _, err := fmt.Fprintf(w, "retry: %d\n: connected\n\n", streamRetryMs); err != nil {
		return
	}
	if err := rc.Flush(); err != nil {
		h.logger.Error(fmt.Sprintf("stream: flush is not supported by the response writer: %s", err))
		return
	}

	events, unsubscribe := h.hub.Subscribe(authCtx.OrganizationID)
	defer unsubscribe()

	heartbeat := time.NewTicker(h.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, open := <-events:
			if !open {
				// Hub отключил медленного клиента или сервер останавливается
				return
			}
			if err := writeStreamEvent(w, ev); err != nil {
				h.logger.Warn(fmt.Sprintf("stream: write %s event: %s", ev.Type, err))
				return
			}
		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				return
			}
		}
		if err := rc.Flush(); err != nil {
			return
		}
	}
}

// writeStreamEvent пишет одно событие в формате SSE: строка event и одна строка data с JSON
func writeStreamEvent(w io.Writer, ev stream.Event) error {
	payload, err := json.Marshal(streamPayload(ev.Data))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, payload)
	return err
}

// streamPayload переводит доменную модель в тот же DTO, что отдаёт REST, —
// фронт разбирает события и ответы одними типами
func streamPayload(data any) any {
	switch v := data.(type) {
	case model.Telemetry:
		return dto.NewTelemetryResponse(v)
	case model.Alert:
		return dto.NewAlertResponse(v)
	case model.Vehicle:
		return dto.NewVehicleResponse(v)
	default:
		// model.Trip уже отдаётся в REST как есть
		return v
	}
}
