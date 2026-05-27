package app

import (
	"context"
	"database/sql"

	storagesqlite "github.com/lqhiyul/personality-type-test/internal/storage/sqlite"
)

func OpenAppDB(ctx context.Context, databasePath string) (*sql.DB, error) {
	return storagesqlite.Open(ctx, databasePath)
}

func RunMigrations(ctx context.Context, db *sql.DB) error {
	return storagesqlite.RunMigrations(ctx, db)
}

func sqliteDSN(databasePath string) string {
	return storagesqlite.DSN(databasePath)
}

func pingSQLite(ctx context.Context, db *sql.DB) error {
	return storagesqlite.Ping(ctx, db)
}
