// Package factory собирает наборы репозиториев, привязанные к конкретной транзакции/соединению.
package factory

import "fleettrack/internal/database"

// RepositoryFactory создаёт набор репозиториев поверх переданного соединения/транзакции
type RepositoryFactory interface {
	New(tx database.DBTX) Repositories
}
