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

// PostgresAlertRuleRepository реализует repository.AlertRuleRepository для PostgreSQL
type PostgresAlertRuleRepository struct {
	db database.DBTX
}

// NewPostgresAlertRuleRepository создаёт новый экземпляр репозитория правил алертов
func NewPostgresAlertRuleRepository(db database.DBTX) *PostgresAlertRuleRepository {
	return &PostgresAlertRuleRepository{
		db: db,
	}
}

// Create создаёт новое правило для алертов в базе данных
// Автоматически заполняет rule.ID и rule.CreatedAt из базы
func (r *PostgresAlertRuleRepository) Create(ctx context.Context, rule *model.AlertRule) error {
	const query = `
	INSERT INTO alert_rules (
		organization_id, 
		type, 
		name, 
		threshold, 
		severity, 
		enabled
	)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		rule.OrganizationID,
		rule.Type,
		rule.Name,
		rule.Threshold,
		rule.Severity,
		rule.Enabled,
	).Scan(&rule.ID, &rule.CreatedAt)
	if err != nil {
		return mapUniqueViolation(err)
	}
	return nil
}

// GetActiveRulesByOrg возвращает все включённые (enabled = true) правила организации
// Это критичный для производительности метод, используемый Alert Engine при обработке телеметрии
func (r *PostgresAlertRuleRepository) GetActiveRulesByOrg(ctx context.Context, orgID int) ([]model.AlertRule, error) {
	const query = `
	SELECT
		id,
		organization_id,
		type, 
		name, 
		threshold, 
		severity, 
		enabled, 
		created_at, 
		updated_at 
	FROM alert_rules
	WHERE organization_id = $1 AND enabled = true`

	rows, err := r.db.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.AlertRule
	for rows.Next() {
		var rule model.AlertRule
		err := rows.Scan(
			&rule.ID,
			&rule.OrganizationID,
			&rule.Type,
			&rule.Name,
			&rule.Threshold,
			&rule.Severity,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

// GetByID возвращает правило по его ID
// Возвращает model.ErrNotFound, если правило с таким ID не существует
func (r *PostgresAlertRuleRepository) GetByID(ctx context.Context, id int) (model.AlertRule, error) {
	const query = `
	SELECT
		id,
		organization_id,
		type, 
		name, 
		threshold, 
		severity, 
		enabled, 
		created_at, 
		updated_at 
	FROM alert_rules
	WHERE id = $1`

	var rule model.AlertRule
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rule.ID,
		&rule.OrganizationID,
		&rule.Type,
		&rule.Name,
		&rule.Threshold,
		&rule.Severity,
		&rule.Enabled,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.AlertRule{}, model.ErrNotFound
		}
		return model.AlertRule{}, err
	}
	return rule, nil
}

// Update обновляет конфигурацию существующего правила и проставляет updated_at = NOW()
// Возвращает обновлённую сущность или model.ErrNotFound, если правило не найдено
func (r *PostgresAlertRuleRepository) Update(ctx context.Context, upd model.AlertRule) (model.AlertRule, error) {
	const query = `
	UPDATE alert_rules
	SET 
		type = $1,
		name = $2,
		threshold = $3,
		severity = $4,
		enabled = $5,
		updated_at = NOW()
	WHERE id = $6
	RETURNING id, organization_id, type, name, threshold, severity, enabled, created_at, updated_at`

	var rule model.AlertRule
	err := r.db.QueryRow(ctx, query,
		upd.Type,
		upd.Name,
		upd.Threshold,
		upd.Severity,
		upd.Enabled,
		upd.ID,
	).Scan(
		&rule.ID,
		&rule.OrganizationID,
		&rule.Type,
		&rule.Name,
		&rule.Threshold,
		&rule.Severity,
		&rule.Enabled,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.AlertRule{}, model.ErrNotFound
		}
		return model.AlertRule{}, err
	}
	return rule, nil
}

// DeleteByID удаляет правило по его ID и возвращает удалённую сущность
// Возвращает model.ErrNotFound, если удаляемая запись не найдена
func (r *PostgresAlertRuleRepository) DeleteByID(ctx context.Context, id int) (model.AlertRule, error) {
	const query = `
	DELETE FROM alert_rules
	WHERE id = $1
	RETURNING id, organization_id, type, name, threshold, severity, enabled, created_at, updated_at`

	var rule model.AlertRule
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rule.ID,
		&rule.OrganizationID,
		&rule.Type,
		&rule.Name,
		&rule.Threshold,
		&rule.Severity,
		&rule.Enabled,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.AlertRule{}, model.ErrNotFound
		}
		return model.AlertRule{}, err
	}
	return rule, nil
}

// buildAlertRuleWhereClause формирует динамический WHERE-блок для фильтрации правил
func buildAlertRuleWhereClause(filter model.AlertRuleFilter) (string, []any) {
	var conditions []string
	var args []any
	argN := 1

	if filter.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argN))
		args = append(args, *filter.OrganizationID)
		argN++
	}
	if filter.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argN))
		args = append(args, *filter.Type)
		argN++
	}
	if filter.Name != nil {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argN))
		args = append(args, "%"+*filter.Name+"%")
		argN++
	}
	if filter.MinThreshold != nil {
		conditions = append(conditions, fmt.Sprintf("threshold >= $%d", argN))
		args = append(args, *filter.MinThreshold)
		argN++
	}
	if filter.MaxThreshold != nil {
		conditions = append(conditions, fmt.Sprintf("threshold <= $%d", argN))
		args = append(args, *filter.MaxThreshold)
		argN++
	}
	if filter.Severity != nil {
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argN))
		args = append(args, *filter.Severity)
		argN++
	}
	if filter.Enabled != nil {
		conditions = append(conditions, fmt.Sprintf("enabled = $%d", argN))
		args = append(args, *filter.Enabled)
		argN++
	}
	if filter.CreatedAt != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argN))
		args = append(args, *filter.CreatedAt)
		argN++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	return whereClause, args
}

// GetList возвращает список правил алертов согласно переданным фильтрам и пагинации
func (r *PostgresAlertRuleRepository) GetList(ctx context.Context, filter model.AlertRuleFilter) ([]model.AlertRule, error) {
	whereClause, args := buildAlertRuleWhereClause(filter)

	query := fmt.Sprintf(`
	SELECT
		id,
		organization_id,
		type, 
		name, 
		threshold, 
		severity, 
		enabled, 
		created_at, 
		updated_at 
	FROM alert_rules
	%s
	ORDER BY id ASC`, whereClause)

	if filter.Limit != nil {
		args = append(args, *filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if filter.Offset != nil {
		args = append(args, *filter.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.AlertRule
	for rows.Next() {
		var rule model.AlertRule
		err := rows.Scan(
			&rule.ID,
			&rule.OrganizationID,
			&rule.Type,
			&rule.Name,
			&rule.Threshold,
			&rule.Severity,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}
