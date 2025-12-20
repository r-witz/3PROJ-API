package database

import (
	"context"
	"errors"
	"time"

	"duskforge-api/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type DB struct {
	Pool *pgxpool.Pool
}

func New(databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		logger.Logger.Error("failed to parse database URL", zap.Error(err))
		return nil, errors.New("failed to parse database URL")
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		logger.Logger.Error("failed to create connection pool", zap.Error(err))
		return nil, errors.New("failed to create connection pool")
	}

	if err := pool.Ping(context.Background()); err != nil {
		logger.Logger.Error("failed to ping database", zap.Error(err))
		return nil, errors.New("failed to ping database")
	}

	logger.Logger.Info("database connection established",
		zap.Int32("max_conns", config.MaxConns),
		zap.Int32("min_conns", config.MinConns),
	)

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
	logger.Logger.Info("database connection closed")
}
