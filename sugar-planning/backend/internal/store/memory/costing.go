package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type costing struct{ s *Store }

// Costing returns the costing repository.
func (s *Store) Costing() store.Costing { return costing{s} }

// ---------------------------------------------------------------------------
// Elements
// ---------------------------------------------------------------------------

func (c costing) ListElements(_ context.Context, opts store.ListOptions) (store.Page[domain.CostElement], error) {
	c.s.lock()
	defer c.s.unlock()

	var items []domain.CostElement
	for _, e := range c.s.d.costElements {
		if opts.Active != nil && e.Active != *opts.Active {
			continue
		}
		if opts.Search != "" &&
			!strings.Contains(strings.ToLower(e.Code+" "+e.Name), strings.ToLower(opts.Search)) {
			continue
		}
		items = append(items, e)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Category != items[j].Category {
			return items[i].Category < items[j].Category
		}
		return items[i].Code < items[j].Code
	})
	return paginate(items, opts.Normalise()), nil
}

func (c costing) GetElement(_ context.Context, id string) (domain.CostElement, error) {
	c.s.lock()
	defer c.s.unlock()
	e, ok := c.s.d.costElements[id]
	if !ok {
		return domain.CostElement{}, fmt.Errorf("%w: cost element %s", domain.ErrNotFound, id)
	}
	return e, nil
}

func (c costing) SaveElement(_ context.Context, e domain.CostElement, actor string) (domain.CostElement, error) {
	c.s.lock()
	defer c.s.unlock()

	// The code is the business key, so a repeat updates in place: the cost
	// structure is configuration, and re-running a seed or an import must be
	// idempotent.
	if e.ID == "" {
		for _, existing := range c.s.d.costElements {
			if existing.Code == e.Code {
				e.ID, e.RowVersion = existing.ID, existing.RowVersion
				e.CreatedAt, e.CreatedBy = existing.CreatedAt, existing.CreatedBy
				break
			}
		}
	}
	if e.ID == "" {
		e.ID = uuid.NewString()
		e.CreatedAt, e.CreatedBy = nowUTC(), actor
		e.RowVersion = 0
	} else if prev, ok := c.s.d.costElements[e.ID]; ok {
		if prev.RowVersion != e.RowVersion {
			return domain.CostElement{}, fmt.Errorf(
				"%w: cost element %s is at version %d, you have %d",
				domain.ErrConflict, prev.Code, prev.RowVersion, e.RowVersion)
		}
		e.CreatedAt, e.CreatedBy = prev.CreatedAt, prev.CreatedBy
	}
	e.UpdatedAt, e.UpdatedBy, e.RowVersion = nowUTC(), actor, e.RowVersion+1
	c.s.d.costElements[e.ID] = e
	return e, nil
}

// ---------------------------------------------------------------------------
// Rates
// ---------------------------------------------------------------------------

func (c costing) ListRates(_ context.Context, f store.CostFilter) ([]domain.CostRate, error) {
	c.s.lock()
	defer c.s.unlock()

	var out []domain.CostRate
	for _, r := range c.s.d.costRates {
		if f.FactoryID != "" && r.FactoryID != f.FactoryID {
			continue
		}
		if f.SeasonID != "" && r.SeasonID != "" && r.SeasonID != f.SeasonID {
			continue
		}
		if f.ElementID != "" && r.ElementID != f.ElementID {
			continue
		}
		if f.RateType != "" && string(r.RateType) != f.RateType {
			continue
		}
		if f.On != "" && !r.IsEffectiveOn(f.On) {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ElementID != out[j].ElementID {
			return out[i].ElementID < out[j].ElementID
		}
		if out[i].RateType != out[j].RateType {
			return out[i].RateType < out[j].RateType
		}
		return out[i].ValidFrom < out[j].ValidFrom
	})
	return out, nil
}

func (c costing) SaveRate(_ context.Context, r domain.CostRate, actor string) (domain.CostRate, error) {
	c.s.lock()
	defer c.s.unlock()

	// The business key is element, factory, type and start date: re-entering a
	// rate for a date that already has one corrects it rather than creating a
	// second rate that silently competes with the first.
	if r.ID == "" {
		for _, existing := range c.s.d.costRates {
			if existing.ElementID == r.ElementID && existing.FactoryID == r.FactoryID &&
				existing.RateType == r.RateType && existing.ValidFrom == r.ValidFrom {
				r.ID, r.RowVersion = existing.ID, existing.RowVersion
				r.CreatedAt, r.CreatedBy = existing.CreatedAt, existing.CreatedBy
				break
			}
		}
	}
	if r.ID == "" {
		r.ID = uuid.NewString()
		r.CreatedAt, r.CreatedBy = nowUTC(), actor
		r.RowVersion = 0
	} else if prev, ok := c.s.d.costRates[r.ID]; ok {
		r.CreatedAt, r.CreatedBy = prev.CreatedAt, prev.CreatedBy
	}
	r.UpdatedAt, r.UpdatedBy, r.RowVersion = nowUTC(), actor, r.RowVersion+1
	c.s.d.costRates[r.ID] = r
	return r, nil
}

func (c costing) DeleteRate(_ context.Context, id string) error {
	c.s.lock()
	defer c.s.unlock()
	if _, ok := c.s.d.costRates[id]; !ok {
		return fmt.Errorf("%w: cost rate %s", domain.ErrNotFound, id)
	}
	delete(c.s.d.costRates, id)
	return nil
}

func (c costing) ListExchangeRates(_ context.Context) ([]domain.ExchangeRate, error) {
	c.s.lock()
	defer c.s.unlock()
	var out []domain.ExchangeRate
	for _, r := range c.s.d.exchangeRates {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FromCurrency != out[j].FromCurrency {
			return out[i].FromCurrency < out[j].FromCurrency
		}
		if out[i].ToCurrency != out[j].ToCurrency {
			return out[i].ToCurrency < out[j].ToCurrency
		}
		return out[i].ValidFrom < out[j].ValidFrom
	})
	return out, nil
}

func (c costing) SaveExchangeRate(_ context.Context, r domain.ExchangeRate, actor string) (domain.ExchangeRate, error) {
	c.s.lock()
	defer c.s.unlock()

	if r.ID == "" {
		for _, existing := range c.s.d.exchangeRates {
			if existing.FromCurrency == r.FromCurrency && existing.ToCurrency == r.ToCurrency &&
				existing.ValidFrom == r.ValidFrom {
				r.ID, r.RowVersion = existing.ID, existing.RowVersion
				r.CreatedAt, r.CreatedBy = existing.CreatedAt, existing.CreatedBy
				break
			}
		}
	}
	if r.ID == "" {
		r.ID = uuid.NewString()
		r.CreatedAt, r.CreatedBy = nowUTC(), actor
		r.RowVersion = 0
	} else if prev, ok := c.s.d.exchangeRates[r.ID]; ok {
		r.CreatedAt, r.CreatedBy = prev.CreatedAt, prev.CreatedBy
	}
	r.UpdatedAt, r.UpdatedBy, r.RowVersion = nowUTC(), actor, r.RowVersion+1
	c.s.d.exchangeRates[r.ID] = r
	return r, nil
}

// ---------------------------------------------------------------------------
// Runs
// ---------------------------------------------------------------------------

func (c costing) ListRuns(_ context.Context, f store.CostFilter) (store.Page[domain.CostRun], error) {
	c.s.lock()
	defer c.s.unlock()

	var items []domain.CostRun
	for _, r := range c.s.d.costRuns {
		if f.SeasonID != "" && r.SeasonID != f.SeasonID {
			continue
		}
		if f.VersionID != "" && r.VersionID != f.VersionID {
			continue
		}
		if f.FactoryID != "" && r.FactoryID != f.FactoryID {
			continue
		}
		// The lines belong to the detail view; a list of runs carries only the
		// totals, which is what a screen showing ten runs actually needs.
		r.Lines = nil
		items = append(items, r)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].From != items[j].From {
			return items[i].From > items[j].From // newest first
		}
		return items[i].Code < items[j].Code
	})
	return paginate(items, store.ListOptions{Skip: f.Skip, Top: orDefaultTop(f.Top)}), nil
}

func (c costing) GetRun(_ context.Context, id string) (domain.CostRun, error) {
	c.s.lock()
	defer c.s.unlock()
	r, ok := c.s.d.costRuns[id]
	if !ok {
		return domain.CostRun{}, fmt.Errorf("%w: cost run %s", domain.ErrNotFound, id)
	}

	// The line's element name, category, driver and unit are read back from the
	// element rather than from the stored line, exactly as the SQL store reads
	// them through its join. A run saved before somebody renamed an element
	// then shows the current name in both implementations instead of one
	// showing the old one.
	lines := make([]domain.CostLine, 0, len(r.Lines))
	for _, line := range r.Lines {
		if element, ok := c.s.d.costElements[line.ElementID]; ok {
			line.ElementCode, line.ElementName = element.Code, element.Name
			line.Category, line.Driver, line.Variable = element.Category, element.Driver, element.Variable
		}
		line.DriverUnit = domain.DriverUnit(line.Driver)
		lines = append(lines, line)
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].Category != lines[j].Category {
			return lines[i].Category < lines[j].Category
		}
		return lines[i].ElementCode < lines[j].ElementCode
	})
	r.Lines = lines
	return r, nil
}

func (c costing) SaveRun(_ context.Context, r domain.CostRun, actor string) (domain.CostRun, error) {
	c.s.lock()
	defer c.s.unlock()

	if r.ID == "" {
		for _, existing := range c.s.d.costRuns {
			if existing.SeasonID == r.SeasonID && existing.Code == r.Code {
				r.ID, r.RowVersion = existing.ID, existing.RowVersion
				r.CreatedAt, r.CreatedBy = existing.CreatedAt, existing.CreatedBy
				break
			}
		}
	}
	if r.ID == "" {
		r.ID = uuid.NewString()
		r.CreatedAt, r.CreatedBy = nowUTC(), actor
		r.RowVersion = 0
	} else if prev, ok := c.s.d.costRuns[r.ID]; ok {
		r.CreatedAt, r.CreatedBy = prev.CreatedAt, prev.CreatedBy
	}
	r.UpdatedAt, r.UpdatedBy, r.RowVersion = nowUTC(), actor, r.RowVersion+1
	stored := r
	stored.Lines = append([]domain.CostLine(nil), r.Lines...)
	c.s.d.costRuns[r.ID] = stored
	return r, nil
}
