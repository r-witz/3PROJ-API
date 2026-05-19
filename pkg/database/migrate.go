package database

import (
	"errors"
	"fmt"
	"os"

	"duskforge-api/pkg/logger"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

func RunMigrations(databaseURL string) error {
	path := os.Getenv("MIGRATIONS_PATH")
	if path == "" {
		path = "/migrations"
	}

	m, err := migrate.New("file://"+path, databaseURL)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer m.Close()

	version, dirty, vErr := m.Version()
	if vErr != nil && !errors.Is(vErr, migrate.ErrNilVersion) {
		return fmt.Errorf("read current migration version: %w", vErr)
	}
	if dirty {
		return fmt.Errorf("schema is dirty at version %d: a previous migration failed mid-way; fix the DB state and run `migrate force %d` before redeploying", version, version)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	newVersion, _, _ := m.Version()
	logger.Logger.Info("migrations applied",
		zap.Uint("from_version", version),
		zap.Uint("to_version", newVersion),
	)
	return nil
}
