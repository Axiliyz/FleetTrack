package postgres

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/model"

	"github.com/jackc/pgx/v5"
)

// PostgresRefreshTokenRepository хранит refresh-токены в PostgreSQL
type PostgresRefreshTokenRepository struct {
	db database.DBTX
}

// NewPostgresRefreshTokenRepository создаёт новый репозиторий refresh-токенов
func NewPostgresRefreshTokenRepository(db database.DBTX) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{
		db: db,
	}
}

// Create сохраняет новый refresh-токен, заполняя ID и CreatedAt
func (r *PostgresRefreshTokenRepository) Create(ctx context.Context, t *model.RefreshToken) error {
	const query = `
	INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
	VALUES ($1, $2, $3)
	RETURNING id, created_at`
	return r.db.QueryRow(ctx, query, t.UserID, t.TokenHash, t.ExpiresAt).Scan(&t.ID, &t.CreatedAt)
}

// GetActiveByHash возвращает токен по хешу, если он не отозван и не истёк
func (r *PostgresRefreshTokenRepository) GetActiveByHash(ctx context.Context, tokenHash string) (model.RefreshToken, error) {
	const query = `
	SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
	FROM refresh_tokens
	WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`
	var t model.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.RefreshToken{}, model.ErrNotFound
		}
		return model.RefreshToken{}, err
	}
	return t, nil
}

// Revoke помечает токен отозванным
func (r *PostgresRefreshTokenRepository) Revoke(ctx context.Context, id int) error {
	const query = `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// RevokeActiveByHash помечает токен отозванным по хэшу
// Предотвращает гонки при обновлении
func (r *PostgresRefreshTokenRepository) RevokeActiveByHash(ctx context.Context, hash string) (model.RefreshToken, error) {
	const query = `
	UPDATE refresh_tokens
	SET revoked_at = NOW()
	WHERE token_hash = $1
	AND revoked_at IS NULL
	AND expires_at > NOW()
	RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at`

	var t model.RefreshToken
	err := r.db.QueryRow(ctx, query, hash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.RefreshToken{}, model.ErrNotFound
		}
		return model.RefreshToken{}, err
	}
	return t, nil
}
