// Package postgres реализует интерфейсы репозиториев для PostgreSQL
package postgres

import (
	"context"
	"fleettrack/internal/database"
	"fleettrack/internal/model"
)

// PostgresUserNotificationChannelRepository реализует repository.NotificationChannelRepository для PostgreSQL
type PostgresUserNotificationChannelRepository struct {
	db database.DBTX
}

// NewUserNotificationChannelRepository создаёт новый экземпляр репозитория каналов уведомлений
func NewUserNotificationChannelRepository(db database.DBTX) *PostgresUserNotificationChannelRepository {
	return &PostgresUserNotificationChannelRepository{
		db: db,
	}
}

// Create привязывает новый канал доставки (Email, Telegram, Webhook) к пользователю
func (r *PostgresUserNotificationChannelRepository) Create(ctx context.Context, ch *model.UserNotificationChannel) error {
	const query = `
	INSERT INTO user_notification_channels (
		user_id, 
		type, 
		config, 
		enabled, 
		min_severity
	)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		ch.UserID,
		ch.Type,
		ch.Config,
		ch.Enabled,
		ch.MinSeverity,
	).Scan(&ch.ID, &ch.CreatedAt)
	if err != nil {
		return mapUniqueViolation(err)
	}

	return nil
}

// GetByUserID возвращает все каналы уведомлений, настроенные указанным пользователем
func (r *PostgresUserNotificationChannelRepository) GetByUserID(ctx context.Context, userID int) ([]model.UserNotificationChannel, error) {
	const query = `
	SELECT 
		id, 
		user_id, 
		type, 
		config, 
		enabled, 
		min_severity, 
		created_at
	FROM user_notification_channels
	WHERE user_id = $1
	ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []model.UserNotificationChannel
	for rows.Next() {
		var ch model.UserNotificationChannel
		err := rows.Scan(
			&ch.ID,
			&ch.UserID,
			&ch.Type,
			&ch.Config,
			&ch.Enabled,
			&ch.MinSeverity,
			&ch.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return channels, nil
}

// allowedMinSeverities возвращает список порогов min_severity, при которых канал должен получить алерт
// Например, если алерт CRITICAL, его должны получить каналы с min_severity: LOW, MEDIUM, HIGH и CRITICAL
func allowedMinSeverities(severity model.SeverityLevel) []string {
	switch severity {
	case model.SeverityLevelCritical:
		return []string{
			string(model.SeverityLevelLow),
			string(model.SeverityLevelMedium),
			string(model.SeverityLevelHigh),
			string(model.SeverityLevelCritical),
		}
	case model.SeverityLevelHigh:
		return []string{
			string(model.SeverityLevelLow),
			string(model.SeverityLevelMedium),
			string(model.SeverityLevelHigh),
		}
	case model.SeverityLevelMedium:
		return []string{
			string(model.SeverityLevelLow),
			string(model.SeverityLevelMedium),
		}
	case model.SeverityLevelLow:
		return []string{
			string(model.SeverityLevelLow),
		}
	default:
		return []string{string(severity)}
	}
}

// GetChannelsForAlert находит все активные каналы пользователей организации,
// которые должны получить уведомление об алерте с заданным уровнем severity
func (r *PostgresUserNotificationChannelRepository) GetChannelsForAlert(
	ctx context.Context,
	orgID int,
	severity model.SeverityLevel,
) ([]model.UserNotificationChannel, error) {
	const query = `
	SELECT 
		unc.id, 
		unc.user_id, 
		unc.type, 
		unc.config, 
		unc.enabled, 
		unc.min_severity, 
		unc.created_at
	FROM user_notification_channels unc
	JOIN users u ON u.id = unc.user_id
	WHERE u.organization_id = $1 
	  AND unc.enabled = true
	  AND unc.min_severity = ANY($2)`

	severities := allowedMinSeverities(severity)
	rows, err := r.db.Query(ctx, query, orgID, severities)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []model.UserNotificationChannel
	for rows.Next() {
		var ch model.UserNotificationChannel
		err := rows.Scan(
			&ch.ID,
			&ch.UserID,
			&ch.Type,
			&ch.Config,
			&ch.Enabled,
			&ch.MinSeverity,
			&ch.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return channels, nil
}

// Delete удаляет канал уведомлений по его ID
// Возвращает model.ErrNotFound, если удаляемый канал не найден
func (r *PostgresUserNotificationChannelRepository) Delete(ctx context.Context, id int) error {
	const query = `DELETE FROM user_notification_channels WHERE id = $1`

	ct, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return model.ErrNotFound
	}

	return nil
}
