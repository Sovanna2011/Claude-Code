// Package database owns the connection pool, the migration runner and the transaction helper
// every write path goes through.
package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the pool so callers depend on this package rather than on pgx directly.
type DB struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func Open(ctx context.Context, url string, log *slog.Logger) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 16
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &DB{pool: pool, log: log}, nil
}

func (d *DB) Close() { d.pool.Close() }

func (d *DB) Pool() *pgxpool.Pool { return d.pool }

func (d *DB) Ping(ctx context.Context) error { return d.pool.Ping(ctx) }

// Querier is what repositories accept: satisfied by both the pool and a transaction, so the same
// repository method works inside a transaction and outside one.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// InTx runs fn inside a transaction, committing on success and rolling back on any error or
// panic. Every write path in the service layer goes through here — a multi-table change that
// half-succeeds would leave the area figures inconsistent, which is exactly what the constraint
// triggers deferred to COMMIT are there to prevent.
func (d *DB) InTx(ctx context.Context, fn func(tx Querier) error) (err error) {
	tx, beginErr := d.pool.Begin(ctx)
	if beginErr != nil {
		return fmt.Errorf("begin: %w", beginErr)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				d.log.Error("rollback failed", "error", rbErr)
			}
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}
	// The deferred constraint triggers fire here, so a rule violation surfaces as a commit error
	// rather than at the statement that caused it.
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// Migrate applies every *.sql file in dir that has not been applied yet, in name order, each in
// its own transaction. It is safe to run on every start-up.
func (d *DB) Migrate(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %q: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	// The very first migration creates the table the runner reads, so its absence is not an error.
	applied := map[string]bool{}
	rows, err := d.pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err == nil {
		for rows.Next() {
			var v string
			if scanErr := rows.Scan(&v); scanErr != nil {
				rows.Close()
				return scanErr
			}
			applied[v] = true
		}
		rows.Close()
	}

	for _, name := range files {
		version := strings.TrimSuffix(name, ".sql")
		if applied[version] {
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			return fmt.Errorf("read %s: %w", name, readErr)
		}
		d.log.Info("applying migration", "version", version)
		if execErr := d.InTx(ctx, func(tx Querier) error {
			_, e := tx.Exec(ctx, string(body))
			return e
		}); execErr != nil {
			return fmt.Errorf("migration %s: %w", version, execErr)
		}
	}
	return nil
}
