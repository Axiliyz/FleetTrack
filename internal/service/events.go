package service

import "fleettrack/internal/stream"

// EventPublisher получает события для живой карты диспетчера.
// Сервисы вызывают его только после коммита: событие о том, чего нет в базе
// (транзакция откатилась), хуже пропущенного — карта покажет то, чего не было
type EventPublisher interface {
	Publish(event stream.Event)
}

// eventSink — встраиваемый в сервисы держатель издателя. Без издателя публикация
// молча ничего не делает, поэтому тесты и фоновые сценарии работают как раньше
type eventSink struct {
	events EventPublisher
}

// SetEventPublisher подключает издателя событий; вызывается один раз при сборке приложения
func (e *eventSink) SetEventPublisher(p EventPublisher) {
	e.events = p
}

func (e *eventSink) publish(orgID int, kind string, data any) {
	if e.events == nil || orgID < 1 {
		return
	}
	e.events.Publish(stream.Event{OrganizationID: orgID, Type: kind, Data: data})
}
