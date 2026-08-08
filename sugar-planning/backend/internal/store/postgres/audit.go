package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type auditRepo struct{ s *Store }

// Audit returns the append-only audit repository.
func (s *Store) Audit() store.Audit { return auditRepo{s} }

func (a auditRepo) Append(ctx context.Context, e domain.AuditEvent) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = nowUTC()
	}
	var before, after any
	if e.Before != "" {
		before = []byte(e.Before)
	}
	if e.After != "" {
		after = []byte(e.After)
	}
	_, err := a.s.q.Exec(ctx, `INSERT INTO audit_events
		(id, occurred_at, actor, action, entity, entity_id, before_state, after_state,
		 reason, correlation_id, source_ip)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		e.ID, e.OccurredAt, e.Actor, e.Action, e.Entity, e.EntityID, before, after,
		e.Reason, e.CorrelationID, e.SourceIP)
	return mapError("audit event", err)
}

func (a auditRepo) List(ctx context.Context, f store.AuditFilter) (store.Page[domain.AuditEvent], error) {
	var where []string
	var args []any
	add := func(col, value string) {
		if value == "" {
			return
		}
		args = append(args, value)
		where = append(where, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	add("entity", f.Entity)
	add("entity_id", f.EntityID)
	add("actor", f.Actor)
	add("action", f.Action)

	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := a.s.q.QueryRow(ctx, "SELECT count(*) FROM audit_events"+clause, args...).Scan(&total); err != nil {
		return store.Page[domain.AuditEvent]{}, mapError("audit event", err)
	}

	top := f.Top
	if top <= 0 {
		top = 100
	}
	args = append(args, top, f.Skip)
	rows, err := a.s.q.Query(ctx, fmt.Sprintf(`SELECT id, occurred_at, actor, action, entity,
		entity_id, before_state, after_state, reason, correlation_id, source_ip
		FROM audit_events%s ORDER BY occurred_at DESC LIMIT $%d OFFSET $%d`,
		clause, len(args)-1, len(args)), args...)
	if err != nil {
		return store.Page[domain.AuditEvent]{}, mapError("audit event", err)
	}
	defer rows.Close()

	items := []domain.AuditEvent{}
	for rows.Next() {
		var e domain.AuditEvent
		var before, after []byte
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.Actor, &e.Action, &e.Entity, &e.EntityID,
			&before, &after, &e.Reason, &e.CorrelationID, &e.SourceIP); err != nil {
			return store.Page[domain.AuditEvent]{}, mapError("audit event", err)
		}
		e.Before, e.After = string(before), string(after)
		items = append(items, e)
	}
	return store.Page[domain.AuditEvent]{Items: items, Count: total}, mapError("audit event", rows.Err())
}

type idempotency struct{ s *Store }

// Idempotency returns the idempotency-key repository.
func (s *Store) Idempotency() store.Idempotency { return idempotency{s} }

// Remember claims the key. The INSERT ... ON CONFLICT DO NOTHING is atomic, so
// two concurrent retries of the same request cannot both be treated as fresh.
func (i idempotency) Remember(ctx context.Context, key, endpoint string, response []byte) (bool, []byte, error) {
	tag, err := i.s.q.Exec(ctx,
		`INSERT INTO idempotency_keys (key, endpoint, response) VALUES ($1,$2,$3)
		 ON CONFLICT (endpoint, key) DO NOTHING`, key, endpoint, response)
	if err != nil {
		return false, nil, mapError("idempotency key", err)
	}
	if tag.RowsAffected() == 1 {
		return true, nil, nil
	}
	var previous []byte
	err = i.s.q.QueryRow(ctx,
		"SELECT response FROM idempotency_keys WHERE endpoint = $1 AND key = $2", endpoint, key).
		Scan(&previous)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, nil, mapError("idempotency key", err)
	}
	return false, previous, nil
}

// Complete attaches the response to a key that has already been claimed.
func (i idempotency) Complete(ctx context.Context, key, endpoint string, response []byte) error {
	tag, err := i.s.q.Exec(ctx,
		"UPDATE idempotency_keys SET response = $1 WHERE endpoint = $2 AND key = $3",
		response, endpoint, key)
	if err != nil {
		return mapError("idempotency key", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: idempotency key %s was never claimed", domain.ErrNotFound, key)
	}
	return nil
}
