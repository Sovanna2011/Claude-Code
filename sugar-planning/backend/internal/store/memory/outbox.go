package memory

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type outbox struct{ s *Store }

// Outbox returns the transactional outbox.
func (s *Store) Outbox() store.Outbox { return outbox{s} }

func (o outbox) Append(_ context.Context, e domain.OutboxEvent) error {
	o.s.lock()
	defer o.s.unlock()
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = nowUTC()
	}
	o.s.d.outbox[e.ID] = e
	return nil
}

func (o outbox) Get(_ context.Context, id string) (domain.OutboxEvent, error) {
	o.s.lock()
	defer o.s.unlock()
	e, ok := o.s.d.outbox[id]
	if !ok {
		return domain.OutboxEvent{}, fmt.Errorf("%w: outbox event %s", domain.ErrNotFound, id)
	}
	return e, nil
}

func (o outbox) Due(_ context.Context, now time.Time, limit int) ([]domain.OutboxEvent, error) {
	o.s.lock()
	defer o.s.unlock()

	var out []domain.OutboxEvent
	for _, e := range o.s.d.outbox {
		if e.IsPublished() || e.IsExhausted() {
			continue
		}
		// An event whose backoff has not elapsed is not due yet.
		if now.Before(e.DueAt()) {
			continue
		}
		out = append(out, e)
	}
	// Oldest first, so a queue that has built up drains in the order the
	// changes happened.
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (o outbox) MarkPublished(_ context.Context, id string, at time.Time) error {
	o.s.lock()
	defer o.s.unlock()
	e, ok := o.s.d.outbox[id]
	if !ok {
		return fmt.Errorf("%w: outbox event %s", domain.ErrNotFound, id)
	}
	if e.IsPublished() {
		return fmt.Errorf("%w: outbox event %s is already published", domain.ErrNotFound, id)
	}
	published := at
	e.PublishedAt = &published
	e.Attempts++
	e.LastError = ""
	o.s.d.outbox[id] = e
	return nil
}

func (o outbox) MarkFailed(_ context.Context, id string, at time.Time, reason string) error {
	o.s.lock()
	defer o.s.unlock()
	e, ok := o.s.d.outbox[id]
	if !ok {
		return fmt.Errorf("%w: outbox event %s", domain.ErrNotFound, id)
	}
	if e.IsPublished() {
		return fmt.Errorf("%w: outbox event %s is already published", domain.ErrNotFound, id)
	}
	e.Attempts++
	e.LastError = reason
	// The next attempt is due relative to this one, so the backoff is a wait
	// between attempts rather than a wait since the event was written.
	e.CreatedAt = at
	o.s.d.outbox[id] = e
	return nil
}

func (o outbox) List(_ context.Context, f store.OutboxFilter) (store.Page[domain.OutboxEvent], error) {
	o.s.lock()
	defer o.s.unlock()

	var items []domain.OutboxEvent
	for _, e := range o.s.d.outbox {
		if f.Topic != "" && string(e.Topic) != f.Topic {
			continue
		}
		if f.Unpublished && e.IsPublished() {
			continue
		}
		if f.Exhausted && !e.IsExhausted() {
			continue
		}
		items = append(items, e)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return paginate(items, store.ListOptions{Skip: f.Skip, Top: orDefaultTop(f.Top)}), nil
}
