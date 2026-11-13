package dto

import (
	"fleettrack/internal/model"
	"time"
)

// TelemetryResponse определяет структуру JSON телеметрии
type TelemetryResponse struct {
	TelemetryID int       `json:"telemetry_id"`
	VehicleID   int       `json:"vehicle_id"`
	DeviceID    int       `json:"device_id"`
	Lat         float64   `json:"lat"`
	Lon         float64   `json:"lon"`
	Fuel        *float32  `json:"fuel"`
	ReceivedAt  time.Time `json:"received_at"`
	TripID      int       `json:"trip_id"`
	DistanceKm  float64   `json:"distance_km"`
	SpeedKmh    float64   `json:"speed_kmh"`
}

// NewTelemetryResponse конвертирует model.Telemetry в TelemetryResponse
func NewTelemetryResponse(t model.Telemetry) TelemetryResponse {
	return TelemetryResponse{
		TelemetryID: t.TelemetryID,
		VehicleID:   t.VehicleID,
		DeviceID:    t.DeviceID,
		Lat:         t.Lat,
		Lon:         t.Lon,
		Fuel:        t.Fuel,
		ReceivedAt:  t.ReceivedAt,
		TripID:      t.TripID,
		DistanceKm:  t.DistanceKm,
		SpeedKmh:    t.SpeedKmh,
	}
}

// NewTelemetryListResponse конвертирует срез телеметрии; пустой срез отдаётся как [], а не null
func NewTelemetryListResponse(items []model.Telemetry) []TelemetryResponse {
	res := make([]TelemetryResponse, 0, len(items))
	for _, t := range items {
		res = append(res, NewTelemetryResponse(t))
	}
	return res
}
