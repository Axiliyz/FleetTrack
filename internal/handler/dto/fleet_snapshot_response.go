package dto

import "fleettrack/internal/model"

// FleetSnapshotResponse — тело GET /fleet/snapshot.
// Все списки всегда массивы, даже пустые: фронт итерирует их без проверок на null
type FleetSnapshotResponse struct {
	Org       OrgResponse         `json:"org"`
	Vehicles  []VehicleResponse   `json:"vehicles"`
	Drivers   []DriverResponse    `json:"drivers"`
	Trips     []model.Trip        `json:"trips"`
	Positions []TelemetryResponse `json:"positions"`
	Alerts    []AlertResponse     `json:"alerts"`
	Rules     []AlertRuleResponse `json:"rules"`
}

// NewFleetSnapshotResponse конвертирует model.FleetSnapshot в ответ API
func NewFleetSnapshotResponse(s model.FleetSnapshot) FleetSnapshotResponse {
	res := FleetSnapshotResponse{
		Org:       NewOrgResponse(s.Org),
		Vehicles:  make([]VehicleResponse, 0, len(s.Vehicles)),
		Drivers:   make([]DriverResponse, 0, len(s.Drivers)),
		Trips:     make([]model.Trip, 0, len(s.Trips)),
		Positions: NewTelemetryListResponse(s.Positions),
		Alerts:    NewAlertListResponse(s.Alerts),
		Rules:     NewAlertRuleListResponse(s.Rules),
	}
	for _, v := range s.Vehicles {
		res.Vehicles = append(res.Vehicles, NewVehicleResponse(v))
	}
	for _, d := range s.Drivers {
		res.Drivers = append(res.Drivers, NewDriverResponse(d))
	}
	res.Trips = append(res.Trips, s.Trips...)
	return res
}
