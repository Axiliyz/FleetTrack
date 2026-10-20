// Package stream раздаёт события парка открытым SSE-соединениям внутри одного процесса
package stream

import "sync"

// subscriberBuffer — сколько событий подписчик может не успеть прочитать, прежде чем считаться медленным
const subscriberBuffer = 64

// Типы событий; совпадают с FleetEvent.type во фронте (web/src/types.ts)
const (
	EventTelemetry = "telemetry"
	EventAlert     = "alert"
	EventTrip      = "trip"
	EventVehicle   = "vehicle"
)

// Event — одно событие для карты: организация-получатель, тип и данные для JSON
type Event struct {
	OrganizationID int
	Type           string
	Data           any
}

// Hub хранит подписчиков по организациям: у каждого открытого браузера свой канал событий
type Hub struct {
	channels map[int]map[chan Event]struct{}
	mu       sync.Mutex
	closed   bool
}

// NewHub создаёт пустой Hub
func NewHub() *Hub {
	return &Hub{
		channels: make(map[int]map[chan Event]struct{}),
	}
}

// Subscribe выдаёт канал событий организации orgID и функцию отписки.
func (h *Hub) Subscribe(orgID int) (<-chan Event, func()) {
	ch := make(chan Event, subscriberBuffer)

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		close(ch)
		return ch, func() {}
	}
	if h.channels[orgID] == nil {
		h.channels[orgID] = make(map[chan Event]struct{})
	}
	h.channels[orgID][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() { h.remove(orgID, ch) })
	}
}

// remove убирает канал из набора организации и закрывает его.
// Канал, которого уже нет в наборе, не трогает — так двойной close невозможен
func (h *Hub) remove(orgID int, ch chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.channels[orgID]
	if !ok {
		return
	}
	if _, ok := subs[ch]; !ok {
		return
	}
	delete(subs, ch)
	close(ch)
	if len(subs) == 0 {
		delete(h.channels, orgID)
	}
}

// Publish раздаёт событие всем подписчикам организации и никогда не ждёт медленного:
// если буфер подписчика полон, канал закрывается, браузер переподключается
func (h *Hub) Publish(event Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs := h.channels[event.OrganizationID]
	for ch := range subs {
		select {
		case ch <- event:
		default:
			delete(subs, ch)
			close(ch)
		}
	}
	if subs != nil && len(subs) == 0 {
		delete(h.channels, event.OrganizationID)
	}
}

// Close закрывает каналы всех подписчиков и больше не принимает новых.
// Вызывается при остановке сервера: SSE-соединения сами не заканчиваются,
// и без этого http.Server.Shutdown ждал бы их до таймаута
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for orgID, subs := range h.channels {
		for ch := range subs {
			close(ch)
		}
		delete(h.channels, orgID)
	}
}

// subscribers возвращает число подписчиков организации; нужно тестам и метрикам
func (h *Hub) subscribers(orgID int) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.channels[orgID])
}
