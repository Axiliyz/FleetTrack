package stream

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// nextUserID выдаёт каждой подписке в тестах своего пользователя, чтобы не упираться в лимит на пользователя
var nextUserID atomic.Int64

// subscribe подписывает нового пользователя на организацию orgID
func subscribe(h *Hub, orgID int) (<-chan Event, func()) {
	ch, unsub, err := h.Subscribe(orgID, int(nextUserID.Add(1)))
	if err != nil {
		panic(err)
	}
	return ch, unsub
}

// within падает, если fn не завершилась за отведённое время — так зависание Publish видно как ошибка, а не как вечный тест
func within(t *testing.T, d time.Duration, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("operation did not finish within %s", d)
	}
}

// drain читает канал до закрытия и возвращает, сколько событий в нём было
func drain(t *testing.T, ch <-chan Event) int {
	t.Helper()
	n := 0
	within(t, time.Second, func() {
		for range ch {
			n++
		}
	})
	return n
}

// TestHub_PublishReachesOnlyOwnOrganization проверяет, что событие получает только своя организация
func TestHub_PublishReachesOnlyOwnOrganization(t *testing.T) {
	h := NewHub(0)
	own1, unsub1 := subscribe(h, 1)
	own2, unsub2 := subscribe(h, 1)
	foreign, unsub3 := subscribe(h, 2)
	defer unsub1()
	defer unsub2()
	defer unsub3()

	h.Publish(Event{OrganizationID: 1, Type: "telemetry", Data: 42})

	for i, ch := range []<-chan Event{own1, own2} {
		select {
		case ev := <-ch:
			if ev.Type != "telemetry" || ev.Data != 42 {
				t.Errorf("subscriber %d got %+v", i, ev)
			}
		default:
			t.Errorf("subscriber %d of organization 1 got nothing", i)
		}
	}
	select {
	case ev := <-foreign:
		t.Fatalf("organization 2 received foreign event %+v", ev)
	default:
	}
}

// TestHub_SlowSubscriberIsDropped проверяет, что переполненный подписчик отключается, а Publish не ждёт его
func TestHub_SlowSubscriberIsDropped(t *testing.T) {
	h := NewHub(0)
	// без defer: если Publish зависнет с замком, отложенная отписка тоже встанет на замке и тест не упадёт, а повиснет
	slow, unsubSlow := subscribe(h, 1)

	within(t, time.Second, func() {
		for i := 0; i < subscriberBuffer+1; i++ {
			h.Publish(Event{OrganizationID: 1, Type: "telemetry", Data: i})
		}
	})

	if got := drain(t, slow); got != subscriberBuffer {
		t.Errorf("slow subscriber got %d events before close, want %d", got, subscriberBuffer)
	}
	if n := h.subscribers(1); n != 0 {
		t.Errorf("subscribers after drop = %d, want 0", n)
	}
	h.mu.Lock()
	_, leaked := h.channels[1]
	h.mu.Unlock()
	if leaked {
		t.Error("empty subscriber set of organization 1 left in hub")
	}
	unsubSlow()
}

// TestHub_SlowSubscriberDoesNotHurtOthers проверяет, что отключение медленного не мешает остальным
func TestHub_SlowSubscriberDoesNotHurtOthers(t *testing.T) {
	h := NewHub(0)
	slow, unsubSlow := subscribe(h, 1)
	fast, unsubFast := subscribe(h, 1)

	received := 0
	within(t, time.Second, func() {
		for i := 0; i < subscriberBuffer*3; i++ {
			h.Publish(Event{OrganizationID: 1, Data: i})
			// быстрый читает сразу, медленный не читает вовсе
			<-fast
			received++
		}
	})

	if received != subscriberBuffer*3 {
		t.Errorf("fast subscriber got %d events, want %d", received, subscriberBuffer*3)
	}
	if got := drain(t, slow); got != subscriberBuffer {
		t.Errorf("slow subscriber got %d events, want %d", got, subscriberBuffer)
	}
	if n := h.subscribers(1); n != 1 {
		t.Errorf("subscribers = %d, want only the fast one", n)
	}
	unsubSlow()
	unsubFast()
}

// TestHub_UnsubscribeIsIdempotent проверяет, что повторная отписка и отписка после вытеснения не паникуют
func TestHub_UnsubscribeIsIdempotent(t *testing.T) {
	h := NewHub(0)

	ch, unsub := subscribe(h, 1)
	unsub()
	unsub()
	if _, ok := <-ch; ok {
		t.Error("channel must be closed after unsubscribe")
	}

	dropped, unsubDropped := subscribe(h, 1)
	for i := 0; i < subscriberBuffer+1; i++ {
		h.Publish(Event{OrganizationID: 1})
	}
	drain(t, dropped)
	unsubDropped() // Publish уже закрыл канал — второй close был бы паникой
}

// TestHub_UnsubscribeKeepsOthers проверяет, что отписка удаляет ровно одного подписчика
func TestHub_UnsubscribeKeepsOthers(t *testing.T) {
	h := NewHub(0)
	_, unsubA := subscribe(h, 1)
	b, unsubB := subscribe(h, 1)
	defer unsubB()

	unsubA()
	if n := h.subscribers(1); n != 1 {
		t.Fatalf("subscribers = %d, want 1", n)
	}
	h.Publish(Event{OrganizationID: 1})
	select {
	case <-b:
	default:
		t.Error("remaining subscriber got nothing")
	}
}

// TestHub_PublishWithoutSubscribers проверяет, что публикация в пустую организацию ничего не создаёт
func TestHub_PublishWithoutSubscribers(t *testing.T) {
	h := NewHub(0)
	h.Publish(Event{OrganizationID: 7})
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.channels) != 0 {
		t.Errorf("hub has %d organizations, want 0", len(h.channels))
	}
}

// TestHub_ConcurrentUse — нагрузка из многих горутин; смысл теста раскрывается под go test -race
func TestHub_ConcurrentUse(t *testing.T) {
	h := NewHub(0)
	var wg sync.WaitGroup

	for org := 1; org <= 4; org++ {
		for s := 0; s < 5; s++ {
			wg.Add(1)
			go func(org int) {
				defer wg.Done()
				ch, unsub := subscribe(h, org)
				for i := 0; i < 20; i++ {
					select {
					case <-ch:
					case <-time.After(time.Millisecond):
					}
				}
				unsub()
			}(org)
		}
		wg.Add(1)
		go func(org int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				h.Publish(Event{OrganizationID: org, Data: i})
			}
		}(org)
	}

	within(t, 5*time.Second, wg.Wait)
	for org := 1; org <= 4; org++ {
		if n := h.subscribers(org); n != 0 {
			t.Errorf("organization %d has %d subscribers after everyone left", org, n)
		}
	}
}

// TestHub_Close проверяет, что остановка закрывает все потоки и не даёт подписаться заново
func TestHub_Close(t *testing.T) {
	h := NewHub(0)
	a, unsubA := subscribe(h, 1)
	b, unsubB := subscribe(h, 2)

	h.Close()
	h.Close() // повторная остановка безопасна

	for i, ch := range []<-chan Event{a, b} {
		if _, ok := <-ch; ok {
			t.Errorf("subscriber %d still open after Close", i)
		}
	}
	unsubA() // отписка после Close не должна паниковать на закрытом канале
	unsubB()

	late, unsubLate := subscribe(h, 1)
	if _, ok := <-late; ok {
		t.Error("subscription after Close must return a closed channel")
	}
	unsubLate()
	h.Publish(Event{OrganizationID: 1}) // публикация после Close — молча никуда
}

// TestHub_LimitPerUser проверяет лимит потоков одного пользователя и что отписка освобождает место
func TestHub_LimitPerUser(t *testing.T) {
	h := NewHub(2)
	const user = 7

	_, unsubA, err := h.Subscribe(1, user)
	if err != nil {
		t.Fatal(err)
	}
	_, unsubB, err := h.Subscribe(1, user)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := h.Subscribe(1, user); !errors.Is(err, ErrTooManySubscriptions) {
		t.Fatalf("third subscription err = %v, want ErrTooManySubscriptions", err)
	}
	// другой пользователь той же организации лимитом первого не ограничен
	_, unsubOther, err := h.Subscribe(1, user+1)
	if err != nil {
		t.Fatalf("other user: %v", err)
	}
	defer unsubOther()

	unsubA()
	unsubA() // повторная отписка не должна освобождать второе место
	_, unsubC, err := h.Subscribe(1, user)
	if err != nil {
		t.Fatalf("after unsubscribe: %v", err)
	}
	if _, _, err := h.Subscribe(1, user); !errors.Is(err, ErrTooManySubscriptions) {
		t.Fatalf("double unsubscribe freed extra slot: err = %v", err)
	}
	unsubB()
	unsubC()
}

// TestHub_SlowSubscriberStillHoldsSlot проверяет, что канал, закрытый Hub за медленность,
// держит место пользователя до отписки: иначе поток, который ещё не вышел из цикла, обходил бы лимит
func TestHub_SlowSubscriberStillHoldsSlot(t *testing.T) {
	h := NewHub(1)
	ch, unsub, err := h.Subscribe(1, 9)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= subscriberBuffer; i++ {
		h.Publish(Event{OrganizationID: 1, Data: i})
	}
	drain(t, ch)

	if _, _, err := h.Subscribe(1, 9); !errors.Is(err, ErrTooManySubscriptions) {
		t.Fatalf("err = %v, место должно освобождаться только отпиской", err)
	}
	unsub()
	if _, _, err := h.Subscribe(1, 9); err != nil {
		t.Fatalf("after unsubscribe: %v", err)
	}
}
