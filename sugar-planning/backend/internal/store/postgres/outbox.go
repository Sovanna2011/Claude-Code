package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type outbox struct{ s *Store }

// Outbox returns the transactional outbox.
func (s *Store) Outbox() store.Outbox { return outbox{s} }

const outboxCols = `id, topic, payload, created_at, published_at, attempts, last_error, correlation_id`

func scanOutbox(r scanner) (domain.OutboxEvent, error) {
	var e domain.OutboxEvent
	var topic string
	var payload []byte
	err := r.Scan(&e.ID, &topic, &payload, &e.CreatedAt, &e.PublishedAt,
		&e.Attempts, &e.LastError, &e.CorrelationID)
	e.Topic, e.Payload = domain.Topic(topic), string(payload)
	return e, err
}

// Append writes an event through whatever querier the caller holds - the pool
// outside a transaction, the transaction inside one. That is the whole point of
// the outbox: an event written in the same transaction as the change it
// describes commits with it or not at all.
func (o outbox) Append(ctx context.Context, e domain.OutboxEvent) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = nowUTC()
	}
	if e.Payload == "" {
		e.Payload = "{}"
	}
	_, err := o.s.q.Exec(ctx, `INSERT INTO outbox_events
		(id, topic, payload, created_at, attempts, last_error, correlation_id)
		VALUES ($1,$2,$3,$4,0,'',$5)`,
		e.ID, string(e.Topic), []byte(e.Payload), e.CreatedAt, e.CorrelationID)
	return mapError("outbox event", err)
}

func (o outbox) Get(ctx context.Context, id string) (domain.OutboxEvent, error) {
	e, err := scanOutbox(o.s.q.QueryRow(ctx,
		`SELECT `+outboxCols+` FROM outbox_events WHERE id = $1`, id))
	return e, mapError("outbox event", err)
}

// Due returns the events whose backoff has elapsed.
//
// The rows are locked and skipped rather than merely selected, so two server
// instances running the dispatcher cannot pick up the same event and deliver it
// twice. A consumer still has to deduplicate - a crash between delivering and
// marking published is unavoidable - but two live instances racing is not
// something to leave to chance.
func (o outbox) Due(ctx context.Context, now time.Time, limit int) ([]domain.OutboxEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := o.s.q.Query(ctx, `SELECT `+outboxCols+` FROM outbox_events
		WHERE published_at IS NULL AND attempts < $1
		  AND created_at + make_interval(secs => CASE WHEN attempts = 0 THEN 0
		        ELSE LEAST(3600, power(2, attempts - 1)) END) <= $2
		ORDER BY created_at
		LIMIT $3
		FOR UPDATE SKIP LOCKED`,
		domain.MaxOutboxAttempts, now, limit)
	if err != nil {
		return nil, mapError("outbox event", err)
	}
	defer rows.Close()

	var out []domain.OutboxEvent
	for rows.Next() {
		e, err := scanOutbox(rows)
		if err != nil {
			return nil, mapError("outbox event", err)
		}
		out = append(out, e)
	}
	return out, mapError("outbox event", rows.Err())
}

func (o outbox) MarkPublished(ctx context.Context, id string, at time.Time) error {
	tag, err := o.s.q.Exec(ctx, `UPDATE outbox_events
		SET published_at = $1, attempts = attempts + 1, last_error = ''
		WHERE id = $2 AND published_at IS NULL`, at, id)
	if err != nil {
		return mapError("outbox event", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: outbox event %s is missing or already published",
			domain.ErrNotFound, id)
	}
	return nil
}

// MarkFailed records an attempt that did not deliver. created_at moves to the
// attempt time so the backoff is a wait between attempts rather than a wait
// since the event was written.
func (o outbox) MarkFailed(ctx context.Context, id string, at time.Time, reason string) error {
	if len(reason) > 2000 {
		reason = reason[:2000]
	}
	tag, err := o.s.q.Exec(ctx, `UPDATE outbox_events
		SET attempts = attempts + 1, last_error = $1, created_at = $2
		WHERE id = $3 AND published_at IS NULL`, reason, at, id)
	if err != nil {
		return mapError("outbox event", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: outbox event %s is missing or already published",
			domain.ErrNotFound, id)
	}
	return nil
}

func (o outbox) List(ctx context.Context, f store.OutboxFilter) (store.Page[domain.OutboxEvent], error) {
	w := &execWhere{}
	if f.Topic != "" {
		w.eq("topic", f.Topic)
	}
	if f.Unpublished {
		w.raw("published_at IS NULL")
	}
	if f.Exhausted {
		w.args = append(w.args, domain.MaxOutboxAttempts)
		w.raw(fmt.Sprintf("published_at IS NULL AND attempts >= $%d", len(w.args)))
	}
	clause := w.sql()

	var total int
	if err := o.s.q.QueryRow(ctx, "SELECT count(*) FROM outbox_events"+clause, w.args...).
		Scan(&total); err != nil {
		return store.Page[domain.OutboxEvent]{}, mapError("outbox event", err)
	}
	limit := w.limit(store.ExecutionFilter{Skip: f.Skip, Top: f.Top})

	rows, err := o.s.q.Query(ctx, "SELECT "+outboxCols+" FROM outbox_events"+clause+
		" ORDER BY created_at DESC"+limit, w.args...)
	if err != nil {
		return store.Page[domain.OutboxEvent]{}, mapError("outbox event", err)
	}
	defer rows.Close()

	items := []domain.OutboxEvent{}
	for rows.Next() {
		e, err := scanOutbox(rows)
		if err != nil {
			return store.Page[domain.OutboxEvent]{}, mapError("outbox event", err)
		}
		items = append(items, e)
	}
	return store.Page[domain.OutboxEvent]{Items: items, Count: total},
		mapError("outbox event", rows.Err())
}
