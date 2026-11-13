package service

import (
	"context"
	"fleettrack/internal/logger"
	"fleettrack/internal/model"
	"fleettrack/internal/stream"
	"sync"
	"testing"
)

// recordingPublisher запоминает опубликованные события
type recordingPublisher struct {
	mu     sync.Mutex
	events []stream.Event
}

func (p *recordingPublisher) Publish(e stream.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, e)
}

func (p *recordingPublisher) types() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	res := make([]string, 0, len(p.events))
	for _, e := range p.events {
		res = append(res, e.Type)
	}
	return res
}

func (p *recordingPublisher) orgs() []int {
	p.mu.Lock()
	defer p.mu.Unlock()
	res := make([]int, 0, len(p.events))
	for _, e := range p.events {
		res = append(res, e.OrganizationID)
	}
	return res
}

func assertEvents(t *testing.T, p *recordingPublisher, wantTypes []string, wantOrg int) {
	t.Helper()
	got := p.types()
	if len(got) != len(wantTypes) {
		t.Fatalf("events = %v, want %v", got, wantTypes)
	}
	for i := range got {
		if got[i] != wantTypes[i] {
			t.Fatalf("events = %v, want %v", got, wantTypes)
		}
	}
	for _, org := range p.orgs() {
		if org != wantOrg {
			t.Fatalf("event for organization %d, want %d", org, wantOrg)
		}
	}
}

func newPublishingTelemetryService(t *testing.T, tripRepo *mockTripRepository) (*TelemetryService, *recordingPublisher) {
	t.Helper()
	repo := &mockRepository{}
	vehicles := &mockVehicleRepository{vehicle: &model.Vehicle{ID: 1, OrganizationID: 5}}
	repoFactory := &fakeRepoFactory{telemetry: repo, trip: tripRepo, vehicle: vehicles}
	motion := &fakeMotionService{data: &model.MotionData{DistanceKm: 1.2, SpeedKmh: 80}}
	svc := NewTelemetryService(repo, logger.NewStdLogger(logger.ErrorLevel), &fakeTxManager{}, repoFactory, motion, nil, testBatchConfig)
	pub := &recordingPublisher{}
	svc.SetEventPublisher(pub)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	svc.StartBatcher(ctx)
	return svc, pub
}

// TestTelemetryService_PublishesPointAndTrip проверяет, что после записи уходят точка и рейс с новой статистикой
func TestTelemetryService_PublishesPointAndTrip(t *testing.T) {
	svc, pub := newPublishingTelemetryService(t, &mockTripRepository{trips: []model.Trip{{ID: 1, Status: model.TripStatusRunning}}})

	point := model.Telemetry{DeviceID: 1, VehicleID: 1, Lat: 55.75, Lon: 37.61, Fuel: float32Ptr(60)}
	if _, err := svc.ProcessTelemetry(context.Background(), point); err != nil {
		t.Fatalf("ProcessTelemetry: %v", err)
	}
	assertEvents(t, pub, []string{stream.EventTelemetry, stream.EventTrip}, 5)

	pub.mu.Lock()
	tel, ok := pub.events[0].Data.(model.Telemetry)
	pub.mu.Unlock()
	if !ok || tel.SpeedKmh != 80 || tel.OrganizationID != 5 {
		t.Errorf("telemetry event data = %#v", pub.events[0].Data)
	}
}

// TestTelemetryService_NoTripEventWithoutTrip проверяет, что без активного рейса уходит только точка
func TestTelemetryService_NoTripEventWithoutTrip(t *testing.T) {
	svc, pub := newPublishingTelemetryService(t, &mockTripRepository{})

	point := model.Telemetry{DeviceID: 1, VehicleID: 1, Lat: 55.75, Lon: 37.61}
	if _, err := svc.ProcessTelemetry(context.Background(), point); err != nil {
		t.Fatalf("ProcessTelemetry: %v", err)
	}
	assertEvents(t, pub, []string{stream.EventTelemetry}, 5)
}

// TestTelemetryService_NothingPublishedOnError проверяет, что отклонённая точка не попадает на карту
func TestTelemetryService_NothingPublishedOnError(t *testing.T) {
	svc, pub := newPublishingTelemetryService(t, &mockTripRepository{})

	bad := model.Telemetry{DeviceID: 1, VehicleID: 1, Lat: 255, Lon: 37.61}
	if _, err := svc.ProcessTelemetry(context.Background(), bad); err == nil {
		t.Fatal("expected validation error")
	}
	assertEvents(t, pub, nil, 0)
}

// TestTripService_Publishes проверяет события рейса и их отсутствие без организации или при ошибке
func TestTripService_Publishes(t *testing.T) {
	org := 5
	pub := &recordingPublisher{}
	svc := newTestTripService(&mockTripRepository{}, &mockVehicleRepository{}, logger.NewStdLogger(logger.ErrorLevel))
	svc.SetEventPublisher(pub)

	if _, err := svc.AssignTrip(context.Background(), 1, 1, &org); err != nil {
		t.Fatalf("AssignTrip: %v", err)
	}
	if _, err := svc.UpdateTrip(context.Background(), 1, model.Trip{Status: model.TripStatusSucceeded}, &org); err != nil {
		t.Fatalf("UpdateTrip: %v", err)
	}
	// без организации непонятно, кому показывать рейс — событие не уходит
	if _, err := svc.AssignTrip(context.Background(), 1, 1, nil); err != nil {
		t.Fatalf("AssignTrip without org: %v", err)
	}
	// вместе с рейсом карта узнаёт и о статусе машины: ON_TRIP при старте, IDLE при завершении
	assertEvents(t, pub, []string{stream.EventTrip, stream.EventVehicle, stream.EventTrip, stream.EventVehicle}, 5)

	failing := newTestTripService(&mockTripRepository{createErr: model.ErrInvalidVehicleID}, &mockVehicleRepository{}, logger.NewStdLogger(logger.ErrorLevel))
	failPub := &recordingPublisher{}
	failing.SetEventPublisher(failPub)
	_, _ = failing.AssignTrip(context.Background(), 1, 1, &org)
	assertEvents(t, failPub, nil, 0)
}

// TestVehicleService_PublishesCreate проверяет событие о новой машине и его отсутствие при ошибке
func TestVehicleService_PublishesCreate(t *testing.T) {
	v := model.Vehicle{OrganizationID: 5, VIN: "XTA21099012345678", NumberPlate: "A123BC77", Model: "КАМАЗ 54901", Status: model.VehicleStatusIdle}

	pub := &recordingPublisher{}
	svc := newTestVehicleService(&mockVehicleRepository{}, logger.NewStdLogger(logger.ErrorLevel))
	svc.SetEventPublisher(pub)
	if _, err := svc.CreateVehicle(context.Background(), v); err != nil {
		t.Fatalf("CreateVehicle: %v", err)
	}
	assertEvents(t, pub, []string{stream.EventVehicle}, 5)

	failPub := &recordingPublisher{}
	failing := newTestVehicleService(&mockVehicleRepository{createErr: model.ErrDuplicateVIN}, logger.NewStdLogger(logger.ErrorLevel))
	failing.SetEventPublisher(failPub)
	_, _ = failing.CreateVehicle(context.Background(), v)
	assertEvents(t, failPub, nil, 0)
}

// TestAlertService_PublishesLifecycle проверяет события открытия, закрытия и принятия алерта
func TestAlertService_PublishesLifecycle(t *testing.T) {
	rules := &mockAlertRuleRepo{activeRules: []model.AlertRule{
		{ID: 1, OrganizationID: 1, Type: model.AlertRuleSpeedExceed, Threshold: 80, Severity: model.SeverityLevelHigh},
	}}
	newSvc := func(repo *mockAlertRepo) (*AlertService, *recordingPublisher) {
		factory := &alertTestRepoFactory{alertRepo: repo, notificationRepo: &mockNotificationRepo{}}
		svc := NewAlertService(repo, rules, &mockChannelRepo{}, &mockNotificationRepo{}, &alertTestTxManager{}, factory, logger.NewStdLogger(logger.ErrorLevel), 10)
		pub := &recordingPublisher{}
		svc.SetEventPublisher(pub)
		return svc, pub
	}

	t.Run("нарушение открывает алерт", func(t *testing.T) {
		svc, pub := newSvc(&mockAlertRepo{getActiveErr: model.ErrNotFound})
		if err := svc.EvaluateTelemetry(context.Background(), model.Telemetry{OrganizationID: 1, VehicleID: 42, SpeedKmh: 95}); err != nil {
			t.Fatal(err)
		}
		assertEvents(t, pub, []string{stream.EventAlert}, 1)
	})

	t.Run("повторное нарушение не публикуется", func(t *testing.T) {
		svc, pub := newSvc(&mockAlertRepo{activeAlert: model.Alert{ID: 77, Status: model.AlertStatusFired}})
		if err := svc.EvaluateTelemetry(context.Background(), model.Telemetry{OrganizationID: 1, VehicleID: 42, SpeedKmh: 95}); err != nil {
			t.Fatal(err)
		}
		assertEvents(t, pub, nil, 0)
	})

	t.Run("норма закрывает алерт", func(t *testing.T) {
		svc, pub := newSvc(&mockAlertRepo{activeAlert: model.Alert{ID: 77, Status: model.AlertStatusFired}})
		if err := svc.EvaluateTelemetry(context.Background(), model.Telemetry{OrganizationID: 1, VehicleID: 42, SpeedKmh: 60}); err != nil {
			t.Fatal(err)
		}
		assertEvents(t, pub, []string{stream.EventAlert}, 1)
	})

	t.Run("принятие публикуется, отказ — нет", func(t *testing.T) {
		svc, pub := newSvc(&mockAlertRepo{})
		actor := model.AuthContext{UserID: 7, OrganizationID: 1, Role: model.UserRoleDispatcher}
		if _, err := svc.AcknowledgeAlert(context.Background(), 5, actor); err != nil {
			t.Fatal(err)
		}
		assertEvents(t, pub, []string{stream.EventAlert}, 1)

		svc2, pub2 := newSvc(&mockAlertRepo{ackErr: model.ErrNotFound})
		_, _ = svc2.AcknowledgeAlert(context.Background(), 5, actor)
		assertEvents(t, pub2, nil, 0)
	})
}
