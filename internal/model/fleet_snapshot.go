package model

// FleetSnapshot — состояние парка организации на момент запроса: всё, что нужно карте диспетчера при подключении.
// Дальше карта живёт на потоке событий, а снимок нужен, чтобы стартовать не с пустого экрана
type FleetSnapshot struct {
	Org       Org
	Vehicles  []Vehicle
	Drivers   []Driver
	Trips     []Trip      // рейсы в пути и начатые за окно смены
	Positions []Telemetry // последняя точка по каждой машине
	Alerts    []Alert     // незакрытые и созданные за окно смены
	Rules     []AlertRule
}
