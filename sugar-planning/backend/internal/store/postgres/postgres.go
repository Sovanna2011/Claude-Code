// Package postgres implements the store interfaces on PostgreSQL.
//
// The repositories are thin: they map rows to entities, enforce optimistic
// concurrency through row_version and let the database enforce the business
// keys and check constraints. All business logic lives in the domain and
// service layers.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// querier is the subset of pgx shared by a pool and a transaction, which lets
// every repository method run unchanged inside or outside a transaction.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface{ Scan(dest ...any) error }

// Store is the PostgreSQL implementation of store.Store.
type Store struct {
	pool *pgxpool.Pool
	q    querier // pool, or the transaction when inside InTx
}

// Config holds the connection settings, all of which come from environment
// variables in deployment.
type Config struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	ConnectTimeout  time.Duration
}

// Open connects and verifies the connection.
func Open(ctx context.Context, cfg Config) (*Store, error) {
	pc, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse database dsn: %w", err)
	}
	if cfg.MaxConns > 0 {
		pc.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		pc.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		pc.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.ConnectTimeout > 0 {
		pc.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	}
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool, q: pool}, nil
}

// Close releases the connection pool.
func (s *Store) Close() error {
	if s.pool != nil {
		s.pool.Close()
	}
	return nil
}

// Ping backs the readiness probe.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// InTx runs fn inside a single database transaction. Nested calls join the
// outer transaction rather than opening a second one.
func (s *Store) InTx(ctx context.Context, fn func(store.Store) error) error {
	if _, alreadyInTx := s.q.(pgx.Tx); alreadyInTx {
		return fn(s)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	child := &Store{pool: s.pool, q: tx}
	if err := fn(child); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			return errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Error and value helpers
// ---------------------------------------------------------------------------

// mapError translates PostgreSQL errors into domain errors so that no SQLSTATE
// or constraint name reaches the client.
func mapError(entity string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s", domain.ErrNotFound, entity)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%w: %s violates %s", domain.ErrDuplicate, entity, friendly(pgErr.ConstraintName))
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: %s references a record that does not exist (%s)",
				domain.ErrValidation, entity, friendly(pgErr.ConstraintName))
		case "23514": // check_violation
			return fmt.Errorf("%w: %s violates %s", domain.ErrValidation, entity, friendly(pgErr.ConstraintName))
		case "23502": // not_null_violation
			return fmt.Errorf("%w: %s requires a value for %s", domain.ErrValidation, entity, pgErr.ColumnName)
		}
	}
	return fmt.Errorf("%s: %w", entity, err)
}

// friendly turns a constraint name into something a planner can read.
func friendly(constraint string) string {
	s := strings.TrimSuffix(constraint, "_ck")
	s = strings.TrimSuffix(s, "_key")
	s = strings.TrimSuffix(s, "_fk")
	return strings.ReplaceAll(s, "_", " ")
}

// nu returns nil for an empty string so that an optional foreign key is stored
// as NULL rather than as an empty uuid.
func nu(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ds dereferences a nullable string column.
func ds(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// nd converts a business date to a nullable date parameter.
func nd(d domain.BusinessDate) any {
	if d == "" {
		return nil
	}
	t, err := d.Time()
	if err != nil {
		return nil
	}
	return t
}

// bd converts a nullable date column to a business date.
func bd(p *time.Time) domain.BusinessDate {
	if p == nil {
		return ""
	}
	return domain.NewBusinessDate(*p)
}

// nowUTC is the timestamp every write stamps on a row.
//
// It is rounded to a microsecond because that is the resolution of a PostgreSQL
// timestamptz. Without the rounding the value a caller gets back from an insert
// is fractionally later than the value stored, so re-reading the row - or
// comparing the creation stamp an update returns against the one the insert
// returned - shows a difference that does not exist.
func nowUTC() time.Time { return time.Now().UTC().Round(time.Microsecond) }

// mustDate converts a non-null date column.
func mustDate(t time.Time) domain.BusinessDate { return domain.NewBusinessDate(t) }

// placeholders builds "$1, $2, ... $n".
func placeholders(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(parts, ", ")
}

// assignments builds "col1 = $1, col2 = $2, ...".
func assignments(cols []string, from int) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = fmt.Sprintf("%s = $%d", c, from+i)
	}
	return strings.Join(parts, ", ")
}

const auditCols = "created_at, created_by, updated_at, updated_by, row_version"
