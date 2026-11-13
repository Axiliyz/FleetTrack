// Package postgres реализует интерфейсы репозиториев для PostgreSQL
package postgres

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/model"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// PostgresAlertRepository реализует repository.AlertRepository для PostgreSQL
type PostgresAlertRepository struct {
	db database.DBTX
}

// NewPostgresAlertRepository создаёт новый экземпляр репозитория алертов
func NewPostgresAlertRepository(db database.DBTX) *PostgresAlertRepository {
	return &PostgresAlertRepository{
		db: db,
	}
}

// GetActiveAlert ищет текущий незакрытый алерт (status != 'RESOLVED') для пары машина + правило
// Использует partial unique index idx_alerts_active_unique
// Возвращает model.ErrNotFound, если активного алерта нет
func (r *PostgresAlertRepository) GetActiveAlert(ctx context.Context, vehicleID, ruleID int) (model.Alert, error) {
	const query = `
	SELECT 
		id, 
		organization_id, 
		vehicle_id, 
		rule_id, 
		type, 
		severity, 
		status, 
		message, 
		value, 
		created_at, 
		acknowledged_at, 
		acknowledged_by, 
		resolved_at, 
		resolved_by
	FROM alerts
	WHERE vehicle_id = $1 AND rule_id = $2 AND status != 'RESOLVED'
	ORDER BY created_at DESC
	LIMIT 1`

	var a model.Alert
	err := r.db.QueryRow(ctx, query, vehicleID, ruleID).Scan(
		&a.ID,
		&a.OrganizationID,
		&a.VehicleID,
		&a.RuleID,
		&a.Type,
		&a.Severity,
		&a.Status,
		&a.Message,
		&a.Value,
		&a.CreatedAt,
		&a.AcknowledgedAt,
		&a.AcknowledgedBy,
		&a.ResolvedAt,
		&a.ResolvedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Alert{}, model.ErrNotFound
		}
		return model.Alert{}, err
	}

	return a, nil
}

// Create сохраняет новый сработавший алерт со статусом FIRED в партиционированную таблицу alerts
// Автоматически заполняет a.ID и a.CreatedAt из базы
func (r *PostgresAlertRepository) Create(ctx context.Context, a *model.Alert) error {
	const query = `
	INSERT INTO alerts (
		organization_id, 
		vehicle_id, 
		rule_id, 
		type, 
		severity, 
		status, 
		message, 
		value
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		a.OrganizationID,
		a.VehicleID,
		a.RuleID,
		a.Type,
		a.Severity,
		a.Status,
		a.Message,
		a.Value,
	).Scan(&a.ID, &a.CreatedAt)
	if err != nil {
		return mapUniqueViolation(err)
	}

	return nil
}

// Resolve переводит активный алерт в статус RESOLVED и фиксирует resolved_at = NOW()
// resolvedBy равен nil при автоматическом закрытии системой по телеметрии
func (r *PostgresAlertRepository) Resolve(ctx context.Context, id int, resolvedBy *int) (model.Alert, error) {
	const query = `
	UPDATE alerts
	SET 
		status = 'RESOLVED',
		resolved_at = NOW(),
		resolved_by = $1
	WHERE id = $2 AND status != 'RESOLVED'
	RETURNING 
		id, 
		organization_id, 
		vehicle_id, 
		rule_id, 
		type, 
		severity, 
		status, 
		message, 
		value, 
		created_at, 
		acknowledged_at, 
		acknowledged_by, 
		resolved_at, 
		resolved_by`

	var a model.Alert
	err := r.db.QueryRow(ctx, query, resolvedBy, id).Scan(
		&a.ID,
		&a.OrganizationID,
		&a.VehicleID,
		&a.RuleID,
		&a.Type,
		&a.Severity,
		&a.Status,
		&a.Message,
		&a.Value,
		&a.CreatedAt,
		&a.AcknowledgedAt,
		&a.AcknowledgedBy,
		&a.ResolvedAt,
		&a.ResolvedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Alert{}, model.ErrNotFound
		}
		return model.Alert{}, err
	}

	return a, nil
}

// ResolveByVehicle закрывает все незакрытые алерты машины. Нужен при удалении машины:
// иначе её DEVICE_OFFLINE так и висел бы открытым, хотя машины уже нет
func (r *PostgresAlertRepository) ResolveByVehicle(ctx context.Context, vehicleID int) ([]model.Alert, error) {
	return r.resolveWhere(ctx, "vehicle_id = $1", vehicleID)
}

// ResolveByRule закрывает все незакрытые алерты правила. Нужен при удалении правила:
// удалённое правило больше не проверяется, и закрыть его алерты по телеметрии некому
func (r *PostgresAlertRepository) ResolveByRule(ctx context.Context, ruleID int) ([]model.Alert, error) {
	return r.resolveWhere(ctx, "rule_id = $1", ruleID)
}

// resolveWhere закрывает системой (resolved_by = NULL) незакрытые алерты по условию cond с одним аргументом
func (r *PostgresAlertRepository) resolveWhere(ctx context.Context, cond string, arg int) ([]model.Alert, error) {
	query := `
	UPDATE alerts
	SET status = 'RESOLVED', resolved_at = NOW()
	WHERE ` + cond + ` AND status != 'RESOLVED'
	RETURNING
		id, organization_id, vehicle_id, rule_id, type, severity, status, message, value,
		created_at, acknowledged_at, acknowledged_by, resolved_at, resolved_by`

	rows, err := r.db.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []model.Alert
	for rows.Next() {
		var a model.Alert
		if err := rows.Scan(
			&a.ID, &a.OrganizationID, &a.VehicleID, &a.RuleID, &a.Type, &a.Severity, &a.Status,
			&a.Message, &a.Value, &a.CreatedAt, &a.AcknowledgedAt, &a.AcknowledgedBy,
			&a.ResolvedAt, &a.ResolvedBy,
		); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

// AcknowledgeAlert переводит алерт из статуса FIRED в ACKNOWLEDGED оператором.
// organization_id стоит в WHERE: доступ проверяется тем же запросом, что и меняет строку
func (r *PostgresAlertRepository) AcknowledgeAlert(ctx context.Context, alertID, userID, organizationID int) (model.Alert, error) {
	const query = `
	UPDATE alerts
	SET 
		status = 'ACKNOWLEDGED',
		acknowledged_at = NOW(),
		acknowledged_by = $1
	WHERE id = $2 AND status = 'FIRED' AND organization_id = $3
	RETURNING 
		id, 
		organization_id, 
		vehicle_id, 
		rule_id, 
		type, 
		severity, 
		status, 
		message, 
		value, 
		created_at, 
		acknowledged_at, 
		acknowledged_by, 
		resolved_at, 
		resolved_by`

	var a model.Alert
	err := r.db.QueryRow(ctx, query, userID, alertID, organizationID).Scan(
		&a.ID,
		&a.OrganizationID,
		&a.VehicleID,
		&a.RuleID,
		&a.Type,
		&a.Severity,
		&a.Status,
		&a.Message,
		&a.Value,
		&a.CreatedAt,
		&a.AcknowledgedAt,
		&a.AcknowledgedBy,
		&a.ResolvedAt,
		&a.ResolvedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Alert{}, model.ErrNotFound
		}
		return model.Alert{}, err
	}

	return a, nil
}

// buildAlertWhereClause динамически собирает условия фильтрации для дашборда алертов
func buildAlertWhereClause(filter model.AlertFilter) (string, []any) {
	var conditions []string
	var args []any
	argN := 1

	if filter.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argN))
		args = append(args, *filter.OrganizationID)
		argN++ //nolint:ineffassign
	}
	if filter.VehicleID != nil {
		conditions = append(conditions, fmt.Sprintf("vehicle_id = $%d", argN))
		args = append(args, *filter.VehicleID)
		argN++ //nolint:ineffassign
	}
	if filter.Severity != nil {
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argN))
		args = append(args, *filter.Severity)
		argN++ //nolint:ineffassign
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argN))
		args = append(args, *filter.Status)
		argN++ //nolint:ineffassign
	}
	if filter.CreatedAt != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argN))
		args = append(args, *filter.CreatedAt)
		argN++ //nolint:ineffassign
	}
	if filter.AcknowledgedBy != nil {
		conditions = append(conditions, fmt.Sprintf("acknowledged_by = $%d", argN))
		args = append(args, *filter.AcknowledgedBy)
		argN++ //nolint:ineffassign
	}
	if filter.ResolvedBy != nil {
		conditions = append(conditions, fmt.Sprintf("resolved_by = $%d", argN))
		args = append(args, *filter.ResolvedBy)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	return whereClause, args
}

// GetList возвращает список алертов по фильтрам (организация, статус, даты)
// Запросы с фильтром по status и organization_id используют индекс idx_alerts_org_status_time
func (r *PostgresAlertRepository) GetList(ctx context.Context, filter model.AlertFilter) ([]model.Alert, error) {
	whereClause, args := buildAlertWhereClause(filter)

	query := fmt.Sprintf(`
	SELECT 
		id, 
		organization_id, 
		vehicle_id, 
		rule_id, 
		type, 
		severity, 
		status, 
		message, 
		value, 
		created_at, 
		acknowledged_at, 
		acknowledged_by, 
		resolved_at, 
		resolved_by
	FROM alerts
	%s
	ORDER BY created_at DESC, id DESC`, whereClause)

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
		args = append(args, filter.Limit, filter.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []model.Alert
	for rows.Next() {
		var a model.Alert
		err := rows.Scan(
			&a.ID,
			&a.OrganizationID,
			&a.VehicleID,
			&a.RuleID,
			&a.Type,
			&a.Severity,
			&a.Status,
			&a.Message,
			&a.Value,
			&a.CreatedAt,
			&a.AcknowledgedAt,
			&a.AcknowledgedBy,
			&a.ResolvedAt,
			&a.ResolvedBy,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return alerts, nil
}

// FindOfflineVehicles ищет автомобили с активными трекерами, не присылавшие телеметрию дольше thresholdMinutes.
// Удалённые машины не ищутся, даже если их привязка трекера почему-то осталась открытой
func (r *PostgresAlertRepository) FindOfflineVehicles(ctx context.Context, thresholdMinutes float64) ([]model.OfflineVehicleInfo, error) {
	const query = `
	SELECT 
		v.id, 
		v.organization_id, 
		da.device_id,
		COALESCE(v.last_telemetry_at, da.started_at) AS last_seen,
		EXTRACT(EPOCH FROM (NOW() - COALESCE(v.last_telemetry_at, da.started_at))) / 60 AS minutes_offline
	FROM device_assignments da
	JOIN vehicles v ON v.id = da.vehicle_id
	WHERE da.ended_at IS NULL
	AND v.status <> 'DELETED'
	AND EXTRACT(EPOCH FROM (NOW() - COALESCE(v.last_telemetry_at, da.started_at))) / 60 >= $1`

	rows, err := r.db.Query(ctx, query, thresholdMinutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.OfflineVehicleInfo
	for rows.Next() {
		var info model.OfflineVehicleInfo
		if err := rows.Scan(
			&info.VehicleID,
			&info.OrganizationID,
			&info.DeviceID,
			&info.LastSeen,
			&info.MinutesOffline,
		); err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	return result, nil
}

// AcquireLock захватывает транзакционную advisory-блокировку по ID машины и правила
func (r *PostgresAlertRepository) AcquireLock(ctx context.Context, vehicleID, ruleID int) error {
	const query = `SELECT pg_advisory_xact_lock($1::int, $2::int)`
	_, err := r.db.Exec(ctx, query, vehicleID, ruleID)
	return err
}

// GetByID возвращает алерт организации по ID; чужой алерт неотличим от несуществующего
func (r *PostgresAlertRepository) GetByID(ctx context.Context, id, organizationID int) (model.Alert, error) {
	const query = `SELECT id, 
		organization_id, 
		vehicle_id, 
		rule_id, 
		type, 
		severity, 
		status, 
		message, 
		value, 
		created_at, 
		acknowledged_at, 
		acknowledged_by, 
		resolved_at, 
		resolved_by
		FROM alerts
		WHERE id = $1 AND organization_id = $2`

	var a model.Alert
	err := r.db.QueryRow(ctx, query, id, organizationID).Scan(
		&a.ID,
		&a.OrganizationID,
		&a.VehicleID,
		&a.RuleID,
		&a.Type,
		&a.Severity,
		&a.Status,
		&a.Message,
		&a.Value,
		&a.CreatedAt,
		&a.AcknowledgedAt,
		&a.AcknowledgedBy,
		&a.ResolvedAt,
		&a.ResolvedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Alert{}, model.ErrNotFound
		}
		return model.Alert{}, err
	}

	return a, nil
}
