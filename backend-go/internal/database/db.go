package database

import (
	"context"
	"fmt"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Open initialise le pool de connexions PostgreSQL
func Open(ctx context.Context) (*pgxpool.Pool, error) {
	databaseURL, err := config.GetEnv[string]("DATABASE_URL")
	if err != nil {
		return nil, fmt.Errorf("impossbile de lire DATABASE_URL: %w", err)
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("configuration BDD invalide: %w", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("échec de création du pool pgx: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("impossible de pinger la BDD: %w", err)
	}

	return pool, nil
}