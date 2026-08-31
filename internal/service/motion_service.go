package service

import (
	"fleettrack/internal/model"
	"math"
	"time"
)

const earthRadius = 6371.0

// MotionService - контракт подсчёта движения
type MotionService interface {
	// Calculate считает пройденное расстояние и скорость за dt
	Calculate(last *model.Telemetry, cur model.Telemetry) (*model.MotionData, error)
}

// MotionServiceImpl - структура сервиса движения
type MotionServiceImpl struct {
}

// NewMotionServiceImpl - конструктор
func NewMotionServiceImpl() *MotionServiceImpl {
	return &MotionServiceImpl{}
}

// ValidateTime проверяет корректность времени
func ValidateTime(last, cur time.Time) bool {
	return cur.After(last)
}

// ValidateCoords проверяет валидность координат
func ValidateCoords(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

// Calculate считает пройденное расстояние и скорость за dt
func (s *MotionServiceImpl) Calculate(last *model.Telemetry, cur model.Telemetry) (*model.MotionData, error) {
	if last == nil {
		return nil, model.ErrNoValue
	}
	if !ValidateTime(last.DeviceTimestamp, cur.DeviceTimestamp) {
		return nil, model.ErrInvalidTime
	}
	if !ValidateCoords(last.Lat, last.Lon) || !ValidateCoords(cur.Lat, cur.Lon) {
		return nil, model.ErrInvalidCoords
	}
	var res model.MotionData
	res.DistanceKm = CalculateDistance(*last, cur)
	hours := cur.DeviceTimestamp.Sub(last.DeviceTimestamp).Hours()
	if hours < 0 {
		return nil, model.ErrCalculating
	}
	res.SpeedKmh = res.DistanceKm / hours
	return &res, nil
}

// CalculateDistance позволяет посчитать пройденное расстояние
func CalculateDistance(last, cur model.Telemetry) float64 {
	lat1 := last.Lat * math.Pi / 180
	lat2 := cur.Lat * math.Pi / 180
	lon1 := last.Lon * math.Pi / 180
	lon2 := cur.Lon * math.Pi / 180
	dLat := lat2 - lat1
	dLon := lon2 - lon1
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}
