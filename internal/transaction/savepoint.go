package transaction

import (
	"context"
	"fleettrack/internal/database"
	"fmt"
)

// savepointName - одно имя на всех: точки сохранения не вложены друг в друга,
// каждая отпускается или откатывается раньше, чем начнётся следующая
const savepointName = "fleettrack_item"

// WithSavepoint выполняет fn внутри точки сохранения открытой транзакции tx.
// Ошибка fn откатывает только изменения fn, транзакция остаётся живой - так одна
// плохая запись пачки не тянет за собой остальные.
//
// Возвращает две ошибки, потому что это разные события:
// fnErr - ошибка самой fn, её отдают тому, чья это запись;
// txErr - сломалась сама транзакция (не выполнился SAVEPOINT/ROLLBACK TO/RELEASE),
// дальше в ней работать нельзя.
func WithSavepoint(ctx context.Context, tx database.DBTX, fn func() error) (fnErr error, txErr error) {
	if _, err := tx.Exec(ctx, "SAVEPOINT "+savepointName); err != nil {
		return nil, fmt.Errorf("savepoint: %w", err)
	}
	if fnErr = fn(); fnErr != nil {
		if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+savepointName); err != nil {
			return fnErr, fmt.Errorf("rollback to savepoint: %w", err)
		}
		return fnErr, nil
	}
	if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT "+savepointName); err != nil {
		return nil, fmt.Errorf("release savepoint: %w", err)
	}
	return nil, nil
}
