package handler

import (
	"context"
	"encoding/json"
	"fleettrack/internal/logger"
	"fleettrack/internal/middleware"
	"fleettrack/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockFleetService struct {
	snap      model.FleetSnapshot
	err       error
	lastActor model.AuthContext
}

func (m *mockFleetService) Snapshot(_ context.Context, actor model.AuthContext) (model.FleetSnapshot, error) {
	m.lastActor = actor
	return m.snap, m.err
}

// TestFleetHandler_HandleGetSnapshot проверяет коды ответа и форму JSON снимка
func TestFleetHandler_HandleGetSnapshot(t *testing.T) {
	fuel := float32(42.5)
	tests := []struct {
		name           string
		authCtx        *model.AuthContext
		snap           model.FleetSnapshot
		serviceErr     error
		expectedStatus int
	}{
		{
			name:    "снимок с данными",
			authCtx: &model.AuthContext{UserID: 1, OrganizationID: 5, Role: model.UserRoleDispatcher},
			snap: model.FleetSnapshot{
				Org:       model.Org{ID: 5, Name: "Северный тракт"},
				Vehicles:  []model.Vehicle{{ID: 3, OrganizationID: 5, NumberPlate: "А123ВС77", Status: model.VehicleStatusOnTrip}},
				Positions: []model.Telemetry{{TelemetryID: 9, VehicleID: 3, Lat: 55.75, Lon: 37.61, Fuel: &fuel, SpeedKmh: 81}},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "пустой парк",
			authCtx:        &model.AuthContext{UserID: 1, OrganizationID: 5, Role: model.UserRoleAnalytic},
			snap:           model.FleetSnapshot{Org: model.Org{ID: 5}},
			expectedStatus: http.StatusOK,
		},
		{name: "нет токена", expectedStatus: http.StatusUnauthorized},
		{
			name:           "роль без доступа",
			authCtx:        &model.AuthContext{UserID: 2, OrganizationID: 5, Role: model.UserRoleDriver},
			serviceErr:     model.ErrForbidden,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockFleetService{snap: tt.snap, err: tt.serviceErr}
			h := NewFleetHandler(svc, logger.NewStdLogger(logger.DebugLevel))

			req := httptest.NewRequest(http.MethodGet, "/fleet/snapshot", nil)
			if tt.authCtx != nil {
				req = req.WithContext(middleware.ContextWithAuth(req.Context(), *tt.authCtx))
			}
			rec := httptest.NewRecorder()
			h.HandleGetSnapshot(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("got status %d, want %d, body: %s", rec.Code, tt.expectedStatus, rec.Body.String())
			}
			if tt.expectedStatus != http.StatusOK {
				return
			}

			var body struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			// контракт с фронтом: web/src/types.ts, FleetSnapshot
			for _, key := range []string{"vehicles", "drivers", "trips", "positions", "alerts", "rules"} {
				raw, ok := body.Data[key]
				if !ok || len(raw) == 0 || raw[0] != '[' {
					t.Errorf("%s = %s, want JSON array", key, raw)
				}
			}
			if _, ok := body.Data["org"]; !ok {
				t.Error("org is missing")
			}
		})
	}
}

// TestFleetHandler_PositionHasCoordinates проверяет, что точка в снимке несёт координаты и топливо — без них карте нечего рисовать
func TestFleetHandler_PositionHasCoordinates(t *testing.T) {
	fuel := float32(42.5)
	svc := &mockFleetService{snap: model.FleetSnapshot{
		Positions: []model.Telemetry{{TelemetryID: 9, VehicleID: 3, Lat: 55.75, Lon: 37.61, Fuel: &fuel}},
	}}
	h := NewFleetHandler(svc, logger.NewStdLogger(logger.DebugLevel))
	req := httptest.NewRequest(http.MethodGet, "/fleet/snapshot", nil)
	req = req.WithContext(middleware.ContextWithAuth(req.Context(), model.AuthContext{UserID: 1, OrganizationID: 5, Role: model.UserRoleAdmin}))
	rec := httptest.NewRecorder()
	h.HandleGetSnapshot(rec, req)

	var body struct {
		Data struct {
			Positions []map[string]any `json:"positions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data.Positions) != 1 {
		t.Fatalf("positions = %v", body.Data.Positions)
	}
	p := body.Data.Positions[0]
	if p["lat"] != 55.75 || p["lon"] != 37.61 || p["fuel"] != 42.5 {
		t.Errorf("position = %v", p)
	}
}
