package model

// MotionData содержит сведения о движении
type MotionData struct {
	DistanceKm float64
	SpeedKmh   float64
	// Outlier - скорость между точками неправдоподобна, это скачок GPS, а не движение.
	// Такой отрезок не идёт ни в пробег, ни в скорость, ни в алерты
	Outlier bool
}

// NewMotionData - конструктор данных о движении
func NewMotionData(d, s float64) *MotionData {
	return &MotionData{
		DistanceKm: d,
		SpeedKmh:   s,
	}
}
