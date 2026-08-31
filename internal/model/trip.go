package model

import "time"

// TripStatus - перечисление состояний статуса рейса
type TripStatus string

const (
	// TripStatusRunning - рейс в процессе
	TripStatusRunning TripStatus = "RUNNING"
	// TripStatusCancelled - рейс отменён
	TripStatusCancelled TripStatus = "CANCELLED"
	// TripStatusSucceeded - рейс успешен
	TripStatusSucceeded TripStatus = "SUCCEEDED"
	// TripStatusSleeping - рейс в ожидании
	TripStatusSleeping TripStatus = "SLEEPING"
	// TripStatusServing - идёт ТО авто
	TripStatusServing TripStatus = "SERVING"
)

// Trip определяет структуру рейса
type Trip struct {
	ID        int        `json:"id"`
	DriverID  int        `json:"driver_id"`
	VehicleID int        `json:"vehicle_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	Status    TripStatus `json:"status"`
}
