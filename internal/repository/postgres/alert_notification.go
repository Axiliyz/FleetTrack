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

// FetchPending блокирует и возвращает пачку готовых к отправке уведомлений
// В хайлоаде использует "FOR UPDATE SKIP LOCKED", чтобы несколько воркеров
// могли одновременно вычитывать задачи из очереди без взаимных блокировок
func (r *PostgresAlertNotificationRepository) FetchPending(ctx context.Context, batchSize int) ([]model.AlertNotification, error) {
	const query = `
	SELECT 
		id, 
		alert_id, 
		channel_id, 
		status, 
		attempts, 
		next_retry_at, 
		sent_at, 
		error, 
		created_at
	FROM alert_notifications
	WHERE status = 'PENDING' 
	  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
	ORDER BY id ASC
	LIMIT $1
	FOR UPDATE SKIP LOCKED`

	rows, err := r.db.Query(ctx, query, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nots []model.AlertNotification
	for rows.Next() {
		var n model.AlertNotification
		err := rows.Scan(
			&n.ID,
			&n.AlertID,
			&n.ChannelID,
			&n.Status,
			&n.Attempts,
			&n.NextRetryAt,
			&n.SentAt,
			&n.Error,
			&n.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		nots = append(nots, n)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return nots, nil
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
