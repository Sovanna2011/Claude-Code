package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// Migrations are embedded in the binary so that a deployment is a single
// artefact: the container that runs the API is the container that can migrate
// the database, and the two can never be out of step.
//
//go:embed migrations/*.sql
var migrationFS embed.FS

// Migration is one versioned change.
type Migration struct {
	Version string
	Name    string
	Up      string
	Down    string
}

// LoadMigrations reads the embedded migration files, pairing each up with its
// down. A missing down file is an error rather than a silent omission: the
// specification requires migrations to be reversible when technically safe,
// and "we forgot" is not one of the safe cases.
func LoadMigrations() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	byVersion := map[string]*Migration{}
	var order []string

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		base := strings.TrimSuffix(name, ".sql")
		direction := "up"
		if strings.HasSuffix(base, ".down") {
			direction = "down"
		}
		base = strings.TrimSuffix(strings.TrimSuffix(base, ".down"), ".up")

		version, label, _ := strings.Cut(base, "_")
		m, ok := byVersion[version]
		if !ok {
			m = &Migration{Version: version, Name: label}
			byVersion[version] = m
			order = append(order, version)
		}
		if direction == "up" {
			m.Up = string(body)
		} else {
			m.Down = string(body)
		}
	}

	sort.Strings(order)
	out := make([]Migration, 0, len(order))
	for _, v := range order {
		m := byVersion[v]
		if m.Up == "" {
			return nil, fmt.Errorf("migration %s has no up script", v)
		}
		if m.Down == "" {
			return nil, fmt.Errorf("migration %s has no down script", v)
		}
		out = append(out, *m)
	}
	return out, nil
}

const migrationTableDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version     text PRIMARY KEY,
	name        text NOT NULL,
	applied_at  timestamptz NOT NULL DEFAULT now(),
	applied_by  text NOT NULL DEFAULT current_user
)`

// Applied lists the versions already applied.
func (s *Store) Applied(ctx context.Context) ([]string, error) {
	if _, err := s.q.Exec(ctx, migrationTableDDL); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}
	rows, err := s.q.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// MigrateUp applies every pending migration. Each migration runs in its own
// transaction together with the row that records it, so an interrupted upgrade
// leaves the database on a known version rather than half way through one.
func (s *Store) MigrateUp(ctx context.Context) ([]string, error) {
	migrations, err := LoadMigrations()
	if err != nil {
		return nil, err
	}
	applied, err := s.Applied(ctx)
	if err != nil {
		return nil, err
	}
	done := map[string]bool{}
	for _, v := range applied {
		done[v] = true
	}

	var ran []string
	for _, m := range migrations {
		if done[m.Version] {
			continue
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return ran, fmt.Errorf("begin migration %s: %w", m.Version, err)
		}
		if _, err := tx.Exec(ctx, m.Up); err != nil {
			_ = tx.Rollback(ctx)
			return ran, fmt.Errorf("apply migration %s (%s): %w", m.Version, m.Name, err)
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", m.Version, m.Name); err != nil {
			_ = tx.Rollback(ctx)
			return ran, fmt.Errorf("record migration %s: %w", m.Version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return ran, fmt.Errorf("commit migration %s: %w", m.Version, err)
		}
		ran = append(ran, m.Version+"_"+m.Name)
	}
	return ran, nil
}

// MigrateDown reverses the most recent n migrations.
func (s *Store) MigrateDown(ctx context.Context, n int) ([]string, error) {
	migrations, err := LoadMigrations()
	if err != nil {
		return nil, err
	}
	applied, err := s.Applied(ctx)
	if err != nil {
		return nil, err
	}
	byVersion := map[string]Migration{}
	for _, m := range migrations {
		byVersion[m.Version] = m
	}

	var reverted []string
	for i := len(applied) - 1; i >= 0 && len(reverted) < n; i-- {
		version := applied[i]
		m, ok := byVersion[version]
		if !ok {
			return reverted, fmt.Errorf("migration %s is recorded as applied but is not in this binary", version)
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return reverted, fmt.Errorf("begin rollback %s: %w", version, err)
		}
		if _, err := tx.Exec(ctx, m.Down); err != nil {
			_ = tx.Rollback(ctx)
			return reverted, fmt.Errorf("revert migration %s (%s): %w", version, m.Name, err)
		}
		if _, err := tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", version); err != nil {
			_ = tx.Rollback(ctx)
			return reverted, fmt.Errorf("unrecord migration %s: %w", version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return reverted, fmt.Errorf("commit rollback %s: %w", version, err)
		}
		reverted = append(reverted, version+"_"+m.Name)
	}
	return reverted, nil
}
