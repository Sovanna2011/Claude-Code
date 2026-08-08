package memory

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type imports struct{ s *Store }

// Imports returns the mapping templates and the staging area.
func (s *Store) Imports() store.Imports { return imports{s} }

func (i imports) ListMappings(_ context.Context, kind string) ([]domain.ImportMapping, error) {
	i.s.lock()
	defer i.s.unlock()

	out := []domain.ImportMapping{}
	for _, m := range i.s.d.importMappings {
		if kind != "" && string(m.Kind) != kind {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Code < out[b].Code })
	return out, nil
}

func (i imports) GetMapping(_ context.Context, id string) (domain.ImportMapping, error) {
	i.s.lock()
	defer i.s.unlock()
	m, ok := i.s.d.importMappings[id]
	if !ok {
		return domain.ImportMapping{}, fmt.Errorf("%w: import mapping %s", domain.ErrNotFound, id)
	}
	return m, nil
}

func (i imports) MappingByCode(_ context.Context, code string) (domain.ImportMapping, error) {
	i.s.lock()
	defer i.s.unlock()
	for _, m := range i.s.d.importMappings {
		if m.Code == code {
			return m, nil
		}
	}
	return domain.ImportMapping{}, fmt.Errorf("%w: import mapping %s", domain.ErrNotFound, code)
}

func (i imports) SaveMapping(_ context.Context, m domain.ImportMapping, actor string) (domain.ImportMapping, error) {
	i.s.lock()
	defer i.s.unlock()

	now := nowUTC()
	for _, existing := range i.s.d.importMappings {
		if existing.Code == m.Code && existing.ID != m.ID {
			return domain.ImportMapping{}, fmt.Errorf("%w: import mapping %s", domain.ErrDuplicate, m.Code)
		}
	}
	if m.ID == "" {
		m.ID = uuid.NewString()
		m.CreatedAt, m.CreatedBy, m.RowVersion = now, actor, 1
	} else {
		stored, ok := i.s.d.importMappings[m.ID]
		if !ok {
			return domain.ImportMapping{}, fmt.Errorf("%w: import mapping %s", domain.ErrNotFound, m.ID)
		}
		if m.RowVersion != 0 && m.RowVersion != stored.RowVersion {
			return domain.ImportMapping{}, fmt.Errorf("%w: mapping %s is at version %d, you have %d",
				domain.ErrConflict, stored.Code, stored.RowVersion, m.RowVersion)
		}
		m.CreatedAt, m.CreatedBy = stored.CreatedAt, stored.CreatedBy
		m.RowVersion = stored.RowVersion + 1
	}
	m.UpdatedAt, m.UpdatedBy = now, actor
	m.Columns = append([]domain.ColumnMapping(nil), m.Columns...)
	i.s.d.importMappings[m.ID] = m
	return m, nil
}

func (i imports) ListJobs(_ context.Context, f store.ImportFilter) (store.Page[domain.ImportJob], error) {
	i.s.lock()
	defer i.s.unlock()

	var items []domain.ImportJob
	for _, j := range i.s.d.importJobs {
		if f.Kind != "" && string(j.Kind) != f.Kind {
			continue
		}
		if f.Status != "" && j.Status != f.Status {
			continue
		}
		if f.VersionID != "" && j.VersionID != f.VersionID {
			continue
		}
		items = append(items, j)
	}
	sort.Slice(items, func(a, b int) bool { return items[a].CreatedAt.After(items[b].CreatedAt) })
	return paginate(items, store.ListOptions{Skip: f.Skip, Top: orDefaultTop(f.Top)}), nil
}

func (i imports) GetJob(_ context.Context, id string) (domain.ImportJob, error) {
	i.s.lock()
	defer i.s.unlock()
	j, ok := i.s.d.importJobs[id]
	if !ok {
		return domain.ImportJob{}, fmt.Errorf("%w: import job %s", domain.ErrNotFound, id)
	}
	return j, nil
}

func (i imports) SaveJob(_ context.Context, j domain.ImportJob, actor string) (domain.ImportJob, error) {
	i.s.lock()
	defer i.s.unlock()

	if j.ID == "" {
		j.ID = uuid.NewString()
		j.CreatedAt, j.CreatedBy = nowUTC(), actor
	} else if stored, ok := i.s.d.importJobs[j.ID]; ok {
		j.CreatedAt, j.CreatedBy = stored.CreatedAt, stored.CreatedBy
	} else {
		return domain.ImportJob{}, fmt.Errorf("%w: import job %s", domain.ErrNotFound, j.ID)
	}
	i.s.d.importJobs[j.ID] = j
	return j, nil
}

func (i imports) SaveRows(_ context.Context, jobID string, rows []domain.ImportRow) error {
	i.s.lock()
	defer i.s.unlock()
	if _, ok := i.s.d.importJobs[jobID]; !ok {
		return fmt.Errorf("%w: import job %s", domain.ErrNotFound, jobID)
	}
	i.s.d.importRows[jobID] = append([]domain.ImportRow(nil), rows...)
	return nil
}

func (i imports) Rows(_ context.Context, jobID string, errorsOnly bool, skip, top int) (store.Page[domain.ImportRow], error) {
	i.s.lock()
	defer i.s.unlock()

	var items []domain.ImportRow
	for _, r := range i.s.d.importRows[jobID] {
		if errorsOnly && r.OK() {
			continue
		}
		items = append(items, r)
	}
	sort.Slice(items, func(a, b int) bool { return items[a].RowNo < items[b].RowNo })
	return paginate(items, store.ListOptions{Skip: skip, Top: orDefaultTop(top)}), nil
}
