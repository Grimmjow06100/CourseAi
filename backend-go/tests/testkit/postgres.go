// Package testkit contains reusable setup helpers for integration and end-to-end tests.
package testkit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/config"
	appdb "github.com/Grimmjow06100/course-ai/backend-go/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenPostgres creates a bounded test context and opens the configured PostgreSQL pool.
// The test is skipped unless COURSE_AI_INTEGRATION_TEST=1 and all resources are closed automatically.
func OpenPostgres(t *testing.T, timeout time.Duration) (context.Context, *pgxpool.Pool) {
	t.Helper()

	if value, _ := config.GetEnv[int]("COURSE_AI_INTEGRATION_TEST"); value != 1 {
		t.Skip("set COURSE_AI_INTEGRATION_TEST=1 to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)

	pool, err := appdb.Open(ctx)
	if err != nil {
		t.Fatalf("open PostgreSQL test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return ctx, pool
}

// BeginRollback starts a transaction that is rolled back automatically after the test.
func BeginRollback(t *testing.T, ctx context.Context, pool *pgxpool.Pool) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin PostgreSQL test transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback PostgreSQL test transaction: %v", err)
		}
	})
	return tx
}
