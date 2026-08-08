package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kss/sugarplan/internal/store"
	"github.com/kss/sugarplan/internal/store/postgres"
	"github.com/kss/sugarplan/internal/store/storetest"
)

// Integration tests run against a real PostgreSQL server. Set TEST_DATABASE_URL
// to enable them; without it they are skipped so that `go test ./...` still
// works on a machine with no database.
//
//	TEST_DATABASE_URL=postgres://user:pass@localhost:5432/sugarplan_test go test ./...
func TestConformance(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration tests")
	}
	ctx := context.Background()

	s, err := postgres.Open(ctx, postgres.Config{DSN: dsn, MaxConns: 4, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if _, err := s.MigrateUp(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open truncation pool: %v", err)
	}
	t.Cleanup(pool.Close)

	storetest.Run(t, func(t *testing.T) store.Store {
		truncateAll(ctx, t, pool)
		return s
	})
}

// truncateAll empties every business table between tests. schema_migrations is
// left alone so the schema is migrated once per run.
func truncateAll(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	rows, err := pool.Query(ctx, `SELECT tablename FROM pg_tables
		WHERE schemaname = 'public' AND tablename <> 'schema_migrations'`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatalf("scan table name: %v", err)
		}
		tables = append(tables, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("list tables: %v", err)
	}
	for _, table := range tables {
		if _, err := pool.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
}
