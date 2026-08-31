package model

// MotionData содержит сведения о движении
type MotionData struct {
	DistanceKm float64
	SpeedKmh   float64
}

// NewMotionData - конструктор данных о движении
func NewMotionData(d, s float64) *MotionData {
	return &MotionData{
		DistanceKm: d,
		SpeedKmh:   s,
	}
}
