package db

import (
	"context"
	"fmt"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var postgresEnumTypes = []string{
	"course_generation_status",
	"generation_pipeline_status",
	"course_language",
	"lesson_type",
	"level",
	"activity_difficulty",
	"exercise_type",
	"quiz_type",
	"generation_job_status",
	"generation_job_kind",
}

func Open(ctx context.Context) (*pgxpool.Pool, error) {
	databaseURL, err := config.GetEnv[string]("DATABASE_URL")
	if err != nil {
		return nil, fmt.Errorf("impossbile de lire DATABASE_URL: %w", err)
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("configuration BDD invalide: %w", err)
	}

	maxConns, err := config.GetEnvWithDefault("DB_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}
	minConns, err := config.GetEnvWithDefault("DB_MIN_CONNS", 1)
	if err != nil {
		return nil, err
	}
	maxConnIdleTime, err := config.GetEnvWithDefault("DB_MAX_CONN_IDLE_TIME", 30*time.Minute)
	if err != nil {
		return nil, err
	}
	maxConnLifetime, err := config.GetEnvWithDefault("DB_MAX_CONN_LIFETIME", time.Hour)
	if err != nil {
		return nil, err
	}
	healthCheckPeriod, err := config.GetEnvWithDefault("DB_HEALTH_CHECK_PERIOD", time.Minute)
	if err != nil {
		return nil, err
	}
	if maxConns <= 0 || minConns < 0 || minConns > maxConns {
		return nil, fmt.Errorf("invalid database pool size: min=%d max=%d", minConns, maxConns)
	}

	poolConfig.MaxConns = int32(maxConns)
	poolConfig.MinConns = int32(minConns)
	poolConfig.MaxConnIdleTime = maxConnIdleTime
	poolConfig.MaxConnLifetime = maxConnLifetime
	poolConfig.HealthCheckPeriod = healthCheckPeriod
	previousAfterConnect := poolConfig.AfterConnect
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if previousAfterConnect != nil {
			if err := previousAfterConnect(ctx, conn); err != nil {
				return err
			}
		}
		return registerPostgresEnumTypes(ctx, conn)
	}

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

func registerPostgresEnumTypes(ctx context.Context, conn *pgx.Conn) error {
	if _, err := conn.LoadTypes(ctx, postgresEnumTypes); err != nil {
		return fmt.Errorf("load PostgreSQL enum types: %w", err)
	}
	return nil
}
