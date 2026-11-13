package postgres

import (
	"context"
	"errors"
	"fleettrack/internal/database"
	"fleettrack/internal/model"
	"time"

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

// refreshTokenColumns - колонки токена в порядке scanRefreshToken
const refreshTokenColumns = `id, user_id, family_id, token_hash, expires_at, revoked_at, created_at` //nolint:gosec // список колонок, а не секрет

func scanRefreshToken(row pgx.Row) (model.RefreshToken, error) {
	var t model.RefreshToken
	err := row.Scan(&t.ID, &t.UserID, &t.FamilyID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.RefreshToken{}, model.ErrNotFound
		}
		return model.RefreshToken{}, err
	}
	return t, nil
}

// Create сохраняет новый refresh-токен, заполняя ID, FamilyID и CreatedAt.
// Пустой FamilyID - первый токен после логина, семью ему выдаёт база
func (r *PostgresRefreshTokenRepository) Create(ctx context.Context, t *model.RefreshToken) error {
	const query = `
	INSERT INTO refresh_tokens (user_id, family_id, token_hash, expires_at)
	VALUES ($1, COALESCE(NULLIF($2, '')::uuid, gen_random_uuid()), $3, $4)
	RETURNING id, family_id, created_at`
	return r.db.QueryRow(ctx, query, t.UserID, t.FamilyID, t.TokenHash, t.ExpiresAt).Scan(&t.ID, &t.FamilyID, &t.CreatedAt)
}

// GetActiveByHash возвращает токен по хешу, если он не отозван и не истёк
func (r *PostgresRefreshTokenRepository) GetActiveByHash(ctx context.Context, tokenHash string) (model.RefreshToken, error) {
	const query = `
	SELECT ` + refreshTokenColumns + `
	FROM refresh_tokens
	WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`
	return scanRefreshToken(r.db.QueryRow(ctx, query, tokenHash))
}

// GetByHash возвращает токен по хешу в любом состоянии: по нему AuthService понимает,
// что пришёл уже отозванный токен
func (r *PostgresRefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (model.RefreshToken, error) {
	const query = `
	SELECT ` + refreshTokenColumns + `
	FROM refresh_tokens
	WHERE token_hash = $1`
	return scanRefreshToken(r.db.QueryRow(ctx, query, tokenHash))
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
	RETURNING ` + refreshTokenColumns
	return scanRefreshToken(r.db.QueryRow(ctx, query, hash))
}

// RevokeFamily отзывает все ещё активные токены семьи
func (r *PostgresRefreshTokenRepository) RevokeFamily(ctx context.Context, familyID string) error {
	const query = `
	UPDATE refresh_tokens SET revoked_at = NOW()
	WHERE family_id = $1 AND revoked_at IS NULL`
	_, err := r.db.Exec(ctx, query, familyID)
	return err
}

// RevokeAllForUser отзывает все активные токены пользователя - например, при его удалении
func (r *PostgresRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID int) error {
	const query = `
	UPDATE refresh_tokens SET revoked_at = NOW()
	WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// DeleteExpired удаляет токены, истёкшие раньше before. Отозванные, но ещё не истёкшие токены
// остаются: по ним ловится повторное использование украденного токена
func (r *PostgresRefreshTokenRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	const query = `DELETE FROM refresh_tokens WHERE expires_at < $1`
	tag, err := r.db.Exec(ctx, query, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
