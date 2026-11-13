package postgres

import (
	"context"
	"fleettrack/internal/database"
	"fleettrack/internal/model"
	"time"
)

// PostgresSnapshotRepository реализует SnapshotRepository
type PostgresSnapshotRepository struct {
	db database.DBTX
}

// NewPostgresSnapshotRepository создаёт репозиторий выборок для снимка парка
func NewPostgresSnapshotRepository(db database.DBTX) *PostgresSnapshotRepository {
	return &PostgresSnapshotRepository{db: db}
}

// LatestPositions возвращает последнюю точку с координатами по каждой машине организации.
//
// Таблица telemetry секционирована по received_at, поэтому DISTINCT ON по всей таблице прочитал бы
// все месяцы. Вместо этого для каждой машины LATERAL берёт одну строку по индексу
// (vehicle_id, received_at DESC), а нижняя граница от vehicles.last_telemetry_at даёт планировщику
// отсечь старые секции прямо во время выполнения. Час запаса — на случай, если last_telemetry_at
// чуть отстал от последней вставленной точки
func (r *PostgresSnapshotRepository) LatestPositions(ctx context.Context, organizationID, limit int) ([]model.Telemetry, error) {
	const query = `
	SELECT
		t.id, t.device_id, v.id, v.organization_id,
		t.latitude, t.longitude, t.fuel,
		t.received_at, t.device_timestamp, t.trip_id, t.distance_km, t.speed_kmh
	FROM vehicles v
	CROSS JOIN LATERAL (
		SELECT id, device_id, latitude, longitude, fuel,
			received_at, device_timestamp, trip_id, distance_km, speed_kmh
		FROM telemetry
		WHERE vehicle_id = v.id
			AND received_at >= v.last_telemetry_at - INTERVAL '1 hour'
			AND latitude IS NOT NULL
			AND longitude IS NOT NULL
		ORDER BY received_at DESC, id DESC
		LIMIT 1
	) t
	WHERE v.organization_id = $1
		AND v.status <> 'DELETED'
		AND v.last_telemetry_at IS NOT NULL
	ORDER BY v.id
	LIMIT $2`

	rows, err := r.db.Query(ctx, query, organizationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.Telemetry
	for rows.Next() {
		var t model.Telemetry
		var tripID *int
		if err := rows.Scan(
			&t.TelemetryID, &t.DeviceID, &t.VehicleID, &t.OrganizationID,
			&t.Lat, &t.Lon, &t.Fuel,
			&t.ReceivedAt, &t.DeviceTimestamp, &tripID, &t.DistanceKm, &t.SpeedKmh,
		); err != nil {
			return nil, err
		}
		if tripID != nil {
			t.TripID = *tripID
		}
		points = append(points, t)
	}
	return points, rows.Err()
}

// ShiftTrips возвращает рейсы в пути и рейсы, начатые не раньше since.
// У trips нет organization_id, организация определяется через водителя — как в GetListTrips
func (r *PostgresSnapshotRepository) ShiftTrips(ctx context.Context, organizationID int, since time.Time, limit int) ([]model.Trip, error) {
	const query = `
	SELECT
		t.id, t.driver_id, t.vehicle_id, t.started_at, t.ended_at, t.status,
		t.distance_km, t.avg_speed_kmh, t.max_speed_kmh, t.telemetry_count
	FROM trips t
	JOIN drivers d ON d.id = t.driver_id
	WHERE d.organization_id = $1
		AND (t.status = 'RUNNING' OR t.started_at >= $2)
	ORDER BY t.id DESC
	LIMIT $3`

	rows, err := r.db.Query(ctx, query, organizationID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []model.Trip
	for rows.Next() {
		var t model.Trip
		if err := rows.Scan(
			&t.ID, &t.DriverID, &t.VehicleID, &t.StartedAt, &t.EndedAt, &t.Status,
			&t.DistanceKm, &t.AvgSpeedKmh, &t.MaxSpeedKmh, &t.TelemetryCount,
		); err != nil {
			return nil, err
		}
		trips = append(trips, t)
	}
	return trips, rows.Err()
}

// ShiftAlerts возвращает незакрытые алерты и все алерты, созданные не раньше since
func (r *PostgresSnapshotRepository) ShiftAlerts(ctx context.Context, organizationID int, since time.Time, limit int) ([]model.Alert, error) {
	const query = `
	SELECT
		id, organization_id, vehicle_id, rule_id, type, severity, status, message, value,
		created_at, acknowledged_at, acknowledged_by, resolved_at, resolved_by
	FROM alerts
	WHERE organization_id = $1
		AND (status <> 'RESOLVED' OR created_at >= $2)
	ORDER BY created_at DESC, id DESC
	LIMIT $3`

	rows, err := r.db.Query(ctx, query, organizationID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []model.Alert
	for rows.Next() {
		var a model.Alert
		if err := rows.Scan(
			&a.ID, &a.OrganizationID, &a.VehicleID, &a.RuleID, &a.Type, &a.Severity, &a.Status, &a.Message, &a.Value,
			&a.CreatedAt, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.ResolvedAt, &a.ResolvedBy,
		); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}
