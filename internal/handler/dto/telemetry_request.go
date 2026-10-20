package dto

import (
	"fleettrack/internal/model"
	"time"
)

// TelemetryRequest определяет структуру DTO запроса
type TelemetryRequest struct {
	DeviceID  int      `json:"device_id"`
	VehicleID int      `json:"vehicle_id"`
	Lat       float64  `json:"lat"`
	Lon       float64  `json:"lon"`
	Fuel      *float32 `json:"fuel"`
	// DeviceTimestamp — время замера по часам терминала. Терминал копит точки, пока нет связи,
	// и досылает их пачкой; скорость считается по этому времени, а не по времени приёма.
	// Если не передано, используется время приёма
	DeviceTimestamp *time.Time `json:"device_timestamp,omitempty" format:"date-time"`
}

// ToDomainModel преобразует TelemetryRequest в доменную модель
func (r *TelemetryRequest) ToDomainModel() model.Telemetry {
	t := model.Telemetry{
		DeviceID:  r.DeviceID,
		VehicleID: r.VehicleID,
		Lat:       r.Lat,
		Lon:       r.Lon,
		Fuel:      r.Fuel,
	}
	if r.DeviceTimestamp != nil {
		t.DeviceTimestamp = *r.DeviceTimestamp
	}
	return t
}
