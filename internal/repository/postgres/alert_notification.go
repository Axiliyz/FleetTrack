// Package postgres реализует интерфейсы репозиториев для PostgreSQL
package postgres

import (
	"context"
	"fleettrack/internal/database"
	"fleettrack/internal/model"
	"fmt"
	"strings"
	"time"
)

// PostgresAlertNotificationRepository реализует repository.AlertNotificationRepository для PostgreSQL
type PostgresAlertNotificationRepository struct {
	db database.DBTX
}

// NewPostgresAlertNotificationRepository создаёт новый экземпляр репозитория очереди уведомлений
func NewPostgresAlertNotificationRepository(db database.DBTX) *PostgresAlertNotificationRepository {
	return &PostgresAlertNotificationRepository{
		db: db,
	}
}

// CreateBatch выполняет массовую вставку задач на отправку уведомлений в таблицу alert_notifications
// Вызывается в рамках транзакции создания алерта (Transactional Outbox)
func (r *PostgresAlertNotificationRepository) CreateBatch(ctx context.Context, nots []model.AlertNotification) error {
	if len(nots) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(nots))
	valueArgs := make([]any, 0, len(nots)*4)

	for i, n := range nots {
		offset := i * 4
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d)", offset+1, offset+2, offset+3, offset+4))
		valueArgs = append(valueArgs, n.AlertID, n.ChannelID, string(model.NotificationStatusPending), 0)
	}

	query := fmt.Sprintf(`
	INSERT INTO alert_notifications (
		alert_id, 
		channel_id, 
		status, 
		attempts
	) 
	VALUES %s`, strings.Join(valueStrings, ", "))

	_, err := r.db.Exec(ctx, query, valueArgs...)
	return err
}

// FetchPending блокирует и возвращает пачку готовых к отправке задач с предзагруженными данными
func (r *PostgresAlertNotificationRepository) FetchPending(ctx context.Context, batchSize int) ([]model.NotificationTask, error) {
	const query = `
	WITH locked_tasks AS (
		SELECT id
		FROM alert_notifications
		WHERE status = 'PENDING' 
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY id ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	),
	updated_tasks AS (
		UPDATE alert_notifications n
		SET next_retry_at = NOW() + INTERVAL '2 minutes'
		FROM locked_tasks lt
		WHERE n.id = lt.id
		RETURNING n.id, n.attempts, n.channel_id, n.alert_id
	)
	SELECT 
		u.id, 
		u.attempts, 
		c.id, 
		c.user_id, 
		c.type, 
		c.config, 
		c.enabled, 
		c.min_severity, 
		c.created_at,
		a.id, 
		a.organization_id, 
		a.vehicle_id, 
		a.rule_id, 
		a.type, 
		a.severity, 
		a.status, 
		a.message, 
		a.value, 
		a.created_at
	FROM updated_tasks u
	JOIN user_notification_channels c ON c.id = u.channel_id
	JOIN alerts a ON a.id = u.alert_id
	ORDER BY u.id ASC`

	rows, err := r.db.Query(ctx, query, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []model.NotificationTask
	for rows.Next() {
		var t model.NotificationTask
		err := rows.Scan(
			&t.ID,
			&t.Attempts,
			&t.Channel.ID,
			&t.Channel.UserID,
			&t.Channel.Type,
			&t.Channel.Config,
			&t.Channel.Enabled,
			&t.Channel.MinSeverity,
			&t.Channel.CreatedAt,
			&t.Alert.ID,
			&t.Alert.OrganizationID,
			&t.Alert.VehicleID,
			&t.Alert.RuleID,
			&t.Alert.Type,
			&t.Alert.Severity,
			&t.Alert.Status,
			&t.Alert.Message,
			&t.Alert.Value,
			&t.Alert.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

// MarkSent переводит задачу в статус SENT и фиксирует время успешной отправки
func (r *PostgresAlertNotificationRepository) MarkSent(ctx context.Context, id int) error {
	const query = `
	UPDATE alert_notifications
	SET 
		status = 'SENT',
		sent_at = NOW(),
		error = NULL
	WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	return err
}

// MarkFailed фиксирует ошибку отправки, увеличивает счётчик попыток и ставит время следующего ретрая
// Если nextRetry == nil (попытки исчерпаны), переводит задачу в статус FAILED
func (r *PostgresAlertNotificationRepository) MarkFailed(ctx context.Context, id int, errMsg string, nextRetry *time.Time) error {
	status := model.NotificationStatusPending
	if nextRetry == nil {
		status = model.NotificationStatusFailed
	}

	const query = `
	UPDATE alert_notifications
	SET 
		status = $1,
		attempts = attempts + 1,
		error = $2,
		next_retry_at = $3
	WHERE id = $4`

	_, err := r.db.Exec(ctx, query, string(status), errMsg, nextRetry, id)
	return err
}
