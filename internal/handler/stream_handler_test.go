package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"fleettrack/internal/stream"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// countingHub — настоящий stream.Hub, который считает живые подписки
type countingHub struct {
	*stream.Hub
	active atomic.Int32
}

func (c *countingHub) Subscribe(orgID int) (<-chan stream.Event, func()) {
	ch, unsub := c.Hub.Subscribe(orgID)
	c.active.Add(1)
	var done atomic.Bool
	return ch, func() {
		if done.CompareAndSwap(false, true) {
			c.active.Add(-1)
		}
		unsub()
	}
}

// waitFor ждёт условия до таймаута — подписка и отписка происходят в горутине сервера
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// startStreamServer поднимает /stream за теми же обёртками, что в роутере, и с короткими таймаутами сервера
func startStreamServer(t *testing.T, hub *countingHub, heartbeat time.Duration, authCtx model.AuthContext) *httptest.Server {
	t.Helper()
	log := logger.NewStdLogger(logger.ErrorLevel)
	h := NewStreamHandler(hub, log)
	h.heartbeat = heartbeat

	withAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(middleware.ContextWithAuth(r.Context(), authCtx)))
		})
	}
	chain := middleware.Metrics(middleware.LogQuery(log)(withAuth(http.HandlerFunc(h.HandleStream))))

	srv := httptest.NewUnstartedServer(chain)
	// короче, чем тест держит поток: если дедлайны не сняты, соединение умрёт раньше
	srv.Config.ReadTimeout = 200 * time.Millisecond
	srv.Config.WriteTimeout = 200 * time.Millisecond
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

type sseMessage struct {
	event string
	data  string
}

// readSSE разбирает поток на сообщения; комментарии (строки с «:») отдаются с event = ":"
// readSSE возвращает заголовки ответа; тело читает и закрывает фоновая горутина
func readSSE(ctx context.Context, t *testing.T, url string) (<-chan sseMessage, http.Header) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // тело закрывает горутина ниже, когда поток кончится
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	out := make(chan sseMessage, 16)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		var msg sseMessage
		for sc.Scan() {
			line := sc.Text()
			switch {
			case line == "":
				if msg != (sseMessage{}) {
					out <- msg
				}
				msg = sseMessage{}
			case strings.HasPrefix(line, ":"):
				out <- sseMessage{event: ":", data: strings.TrimSpace(line[1:])}
			case strings.HasPrefix(line, "event: "):
				msg.event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				msg.data = strings.TrimPrefix(line, "data: ")
			}
		}
	}()
	return out, resp.Header
}

func nextEvent(t *testing.T, msgs <-chan sseMessage) sseMessage {
	t.Helper()
	for {
		select {
		case m, ok := <-msgs:
			if !ok {
				t.Fatal("stream closed before event arrived")
			}
			if m.event != ":" {
				return m
			}
		case <-time.After(2 * time.Second):
			t.Fatal("no event within 2s")
		}
	}
}

// TestStream_DeliversEventsPastServerTimeouts — событие доходит, хотя поток живёт дольше ReadTimeout/WriteTimeout,
// а между хендлером и соединением стоят обёртки Metrics и LogQuery
func TestStream_DeliversEventsPastServerTimeouts(t *testing.T) {
	hub := &countingHub{Hub: stream.NewHub()}
	srv := startStreamServer(t, hub, time.Hour, model.AuthContext{UserID: 1, OrganizationID: 5, Role: model.UserRoleDispatcher})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	msgs, header := readSSE(ctx, t, srv.URL)
	if ct := header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q", ct)
	}
	waitFor(t, "subscription", func() bool { return hub.active.Load() == 1 })

	time.Sleep(600 * time.Millisecond) // втрое дольше таймаутов сервера

	fuel := float32(41.5)
	hub.Publish(stream.Event{OrganizationID: 5, Type: stream.EventTelemetry, Data: model.Telemetry{TelemetryID: 9, VehicleID: 3, Lat: 55.75, Lon: 37.61, Fuel: &fuel, SpeedKmh: 82}})
	hub.Publish(stream.Event{OrganizationID: 7, Type: stream.EventAlert, Data: model.Alert{ID: 1}}) // чужая организация

	m := nextEvent(t, msgs)
	if m.event != stream.EventTelemetry {
		t.Fatalf("event = %q", m.event)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(m.data), &payload); err != nil {
		t.Fatalf("data is not JSON: %q", m.data)
	}
	// данные идут в формате REST-ответа, а не доменной модели: snake_case и lat/lon
	if payload["vehicle_id"] != 3.0 || payload["lat"] != 55.75 || payload["fuel"] != 41.5 || payload["speed_kmh"] != 82.0 {
		t.Errorf("payload = %v", payload)
	}

	select {
	case extra := <-msgs:
		if extra.event != ":" {
			t.Errorf("foreign organization event leaked: %+v", extra)
		}
	case <-time.After(100 * time.Millisecond):
	}
}

// TestStream_Heartbeat проверяет, что молчащий поток регулярно получает комментарий-пинг
func TestStream_Heartbeat(t *testing.T) {
	hub := &countingHub{Hub: stream.NewHub()}
	srv := startStreamServer(t, hub, 30*time.Millisecond, model.AuthContext{UserID: 1, OrganizationID: 5, Role: model.UserRoleAdmin})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	msgs, _ := readSSE(ctx, t, srv.URL)

	pings := 0
	deadline := time.After(2 * time.Second)
	for pings < 2 {
		select {
		case m, ok := <-msgs:
			if !ok {
				t.Fatal("stream closed")
			}
			if m.event == ":" && m.data == "ping" {
				pings++
			}
		case <-deadline:
			t.Fatalf("got %d pings, want 2", pings)
		}
	}
}

// TestStream_ClientDisconnectUnsubscribes проверяет, что закрытая вкладка не оставляет подписку в Hub
func TestStream_ClientDisconnectUnsubscribes(t *testing.T) {
	hub := &countingHub{Hub: stream.NewHub()}
	srv := startStreamServer(t, hub, time.Hour, model.AuthContext{UserID: 1, OrganizationID: 5, Role: model.UserRoleAdmin})

	ctx, cancel := context.WithCancel(context.Background())
	readSSE(ctx, t, srv.URL)
	waitFor(t, "subscription", func() bool { return hub.active.Load() == 1 })

	cancel()
	waitFor(t, "unsubscribe after disconnect", func() bool { return hub.active.Load() == 0 })
}

// TestStream_HubCloseEndsStream проверяет, что остановка сервера (Hub.Close) завершает поток у клиента
func TestStream_HubCloseEndsStream(t *testing.T) {
	hub := &countingHub{Hub: stream.NewHub()}
	srv := startStreamServer(t, hub, time.Hour, model.AuthContext{UserID: 1, OrganizationID: 5, Role: model.UserRoleAdmin})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	msgs, _ := readSSE(ctx, t, srv.URL)
	waitFor(t, "subscription", func() bool { return hub.active.Load() == 1 })

	hub.Close()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-msgs:
			if !ok {
				waitFor(t, "unsubscribe", func() bool { return hub.active.Load() == 0 })
				return
			}
		case <-deadline:
			t.Fatal("stream still open after Hub.Close")
		}
	}
}

// TestStream_RequiresAuth проверяет ответ без аутентификации
func TestStream_RequiresAuth(t *testing.T) {
	h := NewStreamHandler(&countingHub{Hub: stream.NewHub()}, logger.NewStdLogger(logger.ErrorLevel))
	rec := httptest.NewRecorder()
	h.HandleStream(rec, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// TestStreamPayload проверяет перевод доменных моделей в формат REST
func TestStreamPayload(t *testing.T) {
	tests := []struct {
		name string
		in   any
		key  string
	}{
		{"телеметрия", model.Telemetry{TelemetryID: 1}, "telemetry_id"},
		{"алерт", model.Alert{ID: 1}, "rule_id"},
		{"машина", model.Vehicle{ID: 1, NumberPlate: "A123BC77"}, "number_plate"},
		{"рейс", model.Trip{ID: 1}, "telemetry_count"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(streamPayload(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			if _, ok := m[tt.key]; !ok {
				t.Errorf("%s: key %q missing in %s", tt.name, tt.key, raw)
			}
		})
	}
}
