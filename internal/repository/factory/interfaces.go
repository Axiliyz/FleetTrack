package factory

import "fleettrack/internal/database"

type RepositoryFactory interface {
	New(tx database.DBTX) Repositories
}
