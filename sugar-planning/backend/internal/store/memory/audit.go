package memory

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type auditRepo struct{ s *Store }

// Audit returns the append-only audit repository.
func (s *Store) Audit() store.Audit { return auditRepo{s} }

func (a auditRepo) Append(_ context.Context, e domain.AuditEvent) error {
	a.s.lock()
	defer a.s.unlock()
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = nowUTC()
	}
	a.s.d.audit = append(a.s.d.audit, e)
	return nil
}

func (a auditRepo) List(_ context.Context, f store.AuditFilter) (store.Page[domain.AuditEvent], error) {
	a.s.lock()
	defer a.s.unlock()

	var items []domain.AuditEvent
	for _, e := range a.s.d.audit {
		if f.Entity != "" && e.Entity != f.Entity {
			continue
		}
		if f.EntityID != "" && e.EntityID != f.EntityID {
			continue
		}
		if f.Actor != "" && e.Actor != f.Actor {
			continue
		}
		if f.Action != "" && e.Action != f.Action {
			continue
		}
		items = append(items, e)
	}
	// Newest first: the audit log is read from the top.
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt.After(items[j].OccurredAt) })

	top := f.Top
	if top <= 0 {
		top = 100
	}
	return paginate(items, store.ListOptions{Skip: f.Skip, Top: top}), nil
}

type idempotency struct{ s *Store }

// Idempotency returns the idempotency-key repository.
func (s *Store) Idempotency() store.Idempotency { return idempotency{s} }

func (i idempotency) Remember(_ context.Context, key, endpoint string, response []byte) (bool, []byte, error) {
	i.s.lock()
	defer i.s.unlock()
	k := endpoint + "|" + key
	if prev, ok := i.s.d.idem[k]; ok {
		return false, prev, nil
	}
	i.s.d.idem[k] = response
	return true, nil, nil
}

func (i idempotency) Complete(_ context.Context, key, endpoint string, response []byte) error {
	i.s.lock()
	defer i.s.unlock()
	k := endpoint + "|" + key
	if _, ok := i.s.d.idem[k]; !ok {
		return fmt.Errorf("%w: idempotency key %s was never claimed", domain.ErrNotFound, key)
	}
	i.s.d.idem[k] = response
	return nil
}
