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

type planning struct{ s *Store }

// Planning returns the planning repository.
func (s *Store) Planning() store.Planning { return planning{s} }

// ---------------------------------------------------------------------------
// Seasons and versions
// ---------------------------------------------------------------------------

func (p planning) ListSeasons(_ context.Context, opts store.ListOptions) (store.Page[domain.Season], error) {
	opts = opts.Normalise()
	p.s.lock()
	defer p.s.unlock()

	var items []domain.Season
	for _, v := range p.s.d.seasons {
		if opts.ParentID != "" && v.CompanyID != opts.ParentID && v.FactoryID != opts.ParentID {
			continue
		}
		if opts.Search != "" && !strings.Contains(strings.ToLower(v.Code+" "+v.Name), strings.ToLower(opts.Search)) {
			continue
		}
		items = append(items, v)
	}
	sort.Slice(items, func(a, b int) bool { return items[a].Code < items[b].Code })
	return paginate(items, opts), nil
}

func (p planning) GetSeason(_ context.Context, id string) (domain.Season, error) {
	p.s.lock()
	defer p.s.unlock()
	v, ok := p.s.d.seasons[id]
	if !ok {
		return domain.Season{}, fmt.Errorf("%w: season %s", domain.ErrNotFound, id)
	}
	return v, nil
}

func (p planning) SaveSeason(_ context.Context, s domain.Season, actor string) (domain.Season, error) {
	p.s.lock()
	defer p.s.unlock()

	if s.ID == "" {
		for _, existing := range p.s.d.seasons {
			if existing.Code == s.Code && existing.FactoryID == s.FactoryID {
				return domain.Season{}, fmt.Errorf("%w: season %s already exists for this factory",
					domain.ErrDuplicate, s.Code)
			}
		}
		s.ID = uuid.NewString()
		s.CreatedAt, s.CreatedBy = nowUTC(), actor
		s.UpdatedAt, s.UpdatedBy, s.RowVersion = s.CreatedAt, actor, 1
		p.s.d.seasons[s.ID] = s
		return s, nil
	}
	prev, ok := p.s.d.seasons[s.ID]
	if !ok {
		return domain.Season{}, fmt.Errorf("%w: season %s", domain.ErrNotFound, s.ID)
	}
	if prev.RowVersion != s.RowVersion {
		return domain.Season{}, fmt.Errorf("%w: season %s is at version %d", domain.ErrConflict, s.ID, prev.RowVersion)
	}
	s.CreatedAt, s.CreatedBy = prev.CreatedAt, prev.CreatedBy
	s.UpdatedAt, s.UpdatedBy, s.RowVersion = nowUTC(), actor, prev.RowVersion+1
	p.s.d.seasons[s.ID] = s
	return s, nil
}

func (p planning) ListVersions(_ context.Context, seasonID string, opts store.ListOptions) (store.Page[domain.PlanVersion], error) {
	opts = opts.Normalise()
	p.s.lock()
	defer p.s.unlock()

	var items []domain.PlanVersion
	for _, v := range p.s.d.versions {
		if seasonID != "" && v.SeasonID != seasonID {
			continue
		}
		if opts.Search != "" &&
			!strings.Contains(strings.ToLower(v.Code+" "+v.Description), strings.ToLower(opts.Search)) {
			continue
		}
		items = append(items, v)
	}
	sort.Slice(items, func(a, b int) bool { return items[a].VersionNo < items[b].VersionNo })
	return paginate(items, opts), nil
}

func (p planning) GetVersion(_ context.Context, id string) (domain.PlanVersion, error) {
	p.s.lock()
	defer p.s.unlock()
	v, ok := p.s.d.versions[id]
	if !ok {
		return domain.PlanVersion{}, fmt.Errorf("%w: plan version %s", domain.ErrNotFound, id)
	}
	return v, nil
}

func (p planning) SaveVersion(_ context.Context, v domain.PlanVersion, actor string) (domain.PlanVersion, error) {
	p.s.lock()
	defer p.s.unlock()

	if v.ID == "" {
		max := 0
		for _, e := range p.s.d.versions {
			if e.SeasonID != v.SeasonID {
				continue
			}
			if e.Code == v.Code {
				return domain.PlanVersion{}, fmt.Errorf("%w: version %s already exists in this season",
					domain.ErrDuplicate, v.Code)
			}
			if e.VersionNo > max {
				max = e.VersionNo
			}
		}
		if v.VersionNo == 0 {
			v.VersionNo = max + 1
		}
		v.ID = uuid.NewString()
		v.CreatedAt, v.CreatedBy = nowUTC(), actor
		v.UpdatedAt, v.UpdatedBy, v.RowVersion = v.CreatedAt, actor, 1
		p.s.d.versions[v.ID] = v
		return v, nil
	}
	prev, ok := p.s.d.versions[v.ID]
	if !ok {
		return domain.PlanVersion{}, fmt.Errorf("%w: plan version %s", domain.ErrNotFound, v.ID)
	}
	if prev.RowVersion != v.RowVersion {
		return domain.PlanVersion{}, fmt.Errorf("%w: plan version %s was changed by %s (version %d, you have %d)",
			domain.ErrConflict, v.Code, prev.UpdatedBy, prev.RowVersion, v.RowVersion)
	}
	v.CreatedAt, v.CreatedBy = prev.CreatedAt, prev.CreatedBy
	v.UpdatedAt, v.UpdatedBy, v.RowVersion = nowUTC(), actor, prev.RowVersion+1
	p.s.d.versions[v.ID] = v
	return v, nil
}

// ---------------------------------------------------------------------------
// Assumptions and product mix
// ---------------------------------------------------------------------------

func (p planning) ListAssumptions(_ context.Context, versionID string) ([]domain.PlanAssumption, error) {
	p.s.lock()
	defer p.s.unlock()
	var out []domain.PlanAssumption
	for _, a := range p.s.d.assumptions {
		if a.VersionID == versionID {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func (p planning) SaveAssumption(_ context.Context, a domain.PlanAssumption, actor string) (domain.PlanAssumption, error) {
	p.s.lock()
	defer p.s.unlock()

	if a.ID == "" {
		for _, e := range p.s.d.assumptions {
			if e.VersionID == a.VersionID && e.Code == a.Code && e.ValidFrom == a.ValidFrom {
				// Same business key: update in place rather than duplicate.
				a.ID = e.ID
				a.RowVersion = e.RowVersion
				a.CreatedAt, a.CreatedBy = e.CreatedAt, e.CreatedBy
				break
			}
		}
	}
	if a.ID == "" {
		a.ID = uuid.NewString()
		a.CreatedAt, a.CreatedBy = nowUTC(), actor
		a.RowVersion = 0
	}
	a.UpdatedAt, a.UpdatedBy, a.RowVersion = nowUTC(), actor, a.RowVersion+1
	p.s.d.assumptions[a.ID] = a
	return a, nil
}

func (p planning) DeleteAssumption(_ context.Context, id string) error {
	p.s.lock()
	defer p.s.unlock()
	if _, ok := p.s.d.assumptions[id]; !ok {
		return fmt.Errorf("%w: assumption %s", domain.ErrNotFound, id)
	}
	delete(p.s.d.assumptions, id)
	return nil
}

func (p planning) ListMix(_ context.Context, versionID string) ([]domain.ProductMixEntry, error) {
	p.s.lock()
	defer p.s.unlock()
	var out []domain.ProductMixEntry
	for _, m := range p.s.d.mix {
		if m.VersionID == versionID {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProductID < out[j].ProductID })
	return out, nil
}

func (p planning) SaveMix(_ context.Context, m domain.ProductMixEntry, actor string) (domain.ProductMixEntry, error) {
	p.s.lock()
	defer p.s.unlock()

	if m.ID == "" {
		for _, e := range p.s.d.mix {
			if e.VersionID == m.VersionID && e.ProductID == m.ProductID && e.PackagingID == m.PackagingID {
				m.ID, m.RowVersion = e.ID, e.RowVersion
				m.CreatedAt, m.CreatedBy = e.CreatedAt, e.CreatedBy
				break
			}
		}
	}
	if m.ID == "" {
		m.ID = uuid.NewString()
		m.CreatedAt, m.CreatedBy = nowUTC(), actor
		m.RowVersion = 0
	}
	m.UpdatedAt, m.UpdatedBy, m.RowVersion = nowUTC(), actor, m.RowVersion+1
	p.s.d.mix[m.ID] = m
	return m, nil
}

func (p planning) DeleteMix(_ context.Context, id string) error {
	p.s.lock()
	defer p.s.unlock()
	if _, ok := p.s.d.mix[id]; !ok {
		return fmt.Errorf("%w: product mix entry %s", domain.ErrNotFound, id)
	}
	delete(p.s.d.mix, id)
	return nil
}

// ---------------------------------------------------------------------------
// Daily facts
// ---------------------------------------------------------------------------

// Natural keys. Re-running an import or regenerating a plan overwrites the row
// with the same key instead of creating a duplicate.

func caneKey(r domain.DailyCanePlan) string {
	return strings.Join([]string{r.VersionID, r.FactoryID, string(r.BusinessDate), r.ShiftID, string(r.Series)}, "|")
}

func productKey(r domain.DailyProductPlan) string {
	return strings.Join([]string{r.VersionID, r.FactoryID, r.LineID, string(r.BusinessDate), r.ShiftID,
		r.ProductID, r.PackagingID, string(r.Series)}, "|")
}

func storageKey(r domain.DailyStoragePlan) string {
	return strings.Join([]string{r.VersionID, r.WarehouseID, r.ProductID, string(r.BusinessDate), string(r.Series)}, "|")
}

func shipmentKey(r domain.DailyShipmentPlan) string {
	return strings.Join([]string{r.VersionID, r.WarehouseID, r.ProductID, r.ChannelID,
		string(r.BusinessDate), string(r.Series)}, "|")
}

func matchIn(list []string, value string) bool {
	if len(list) == 0 {
		return true
	}
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func inDateRange(f store.PlanFilter, d domain.BusinessDate) bool {
	if f.From != "" && d < f.From {
		return false
	}
	if f.To != "" && d > f.To {
		return false
	}
	return true
}

func (p planning) ListCane(_ context.Context, f store.PlanFilter) ([]domain.DailyCanePlan, error) {
	p.s.lock()
	defer p.s.unlock()
	var out []domain.DailyCanePlan
	for _, r := range p.s.d.cane {
		if !matchIn(f.VersionIDs, r.VersionID) || !inDateRange(f, r.BusinessDate) {
			continue
		}
		if f.FactoryID != "" && r.FactoryID != f.FactoryID {
			continue
		}
		if f.Series != "" && r.Series != f.Series {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return caneKey(out[i]) < caneKey(out[j]) })
	return out, nil
}

func (p planning) UpsertCane(_ context.Context, rows []domain.DailyCanePlan, actor string) (int, error) {
	p.s.lock()
	defer p.s.unlock()
	for _, r := range rows {
		k := caneKey(r)
		if prev, ok := p.s.d.cane[k]; ok {
			r.ID, r.CreatedAt, r.CreatedBy, r.RowVersion = prev.ID, prev.CreatedAt, prev.CreatedBy, prev.RowVersion
		} else {
			r.ID, r.CreatedAt, r.CreatedBy = uuid.NewString(), nowUTC(), actor
		}
		r.UpdatedAt, r.UpdatedBy, r.RowVersion = nowUTC(), actor, r.RowVersion+1
		p.s.d.cane[k] = r
	}
	return len(rows), nil
}

func (p planning) ListProducts(_ context.Context, f store.PlanFilter) ([]domain.DailyProductPlan, error) {
	p.s.lock()
	defer p.s.unlock()
	var out []domain.DailyProductPlan
	for _, r := range p.s.d.prodPlans {
		if !matchIn(f.VersionIDs, r.VersionID) || !inDateRange(f, r.BusinessDate) {
			continue
		}
		if !matchIn(f.ProductIDs, r.ProductID) || !matchIn(f.LineIDs, r.LineID) {
			continue
		}
		if f.FactoryID != "" && r.FactoryID != f.FactoryID {
			continue
		}
		if f.Series != "" && r.Series != f.Series {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return productKey(out[i]) < productKey(out[j]) })
	return out, nil
}

func (p planning) UpsertProducts(_ context.Context, rows []domain.DailyProductPlan, actor string) (int, error) {
	p.s.lock()
	defer p.s.unlock()
	for _, r := range rows {
		k := productKey(r)
		if prev, ok := p.s.d.prodPlans[k]; ok {
			r.ID, r.CreatedAt, r.CreatedBy, r.RowVersion = prev.ID, prev.CreatedAt, prev.CreatedBy, prev.RowVersion
		} else {
			r.ID, r.CreatedAt, r.CreatedBy = uuid.NewString(), nowUTC(), actor
		}
		r.UpdatedAt, r.UpdatedBy, r.RowVersion = nowUTC(), actor, r.RowVersion+1
		p.s.d.prodPlans[k] = r
	}
	return len(rows), nil
}

func (p planning) ListStorage(_ context.Context, f store.PlanFilter) ([]domain.DailyStoragePlan, error) {
	p.s.lock()
	defer p.s.unlock()
	var out []domain.DailyStoragePlan
	for _, r := range p.s.d.storage {
		if !matchIn(f.VersionIDs, r.VersionID) || !inDateRange(f, r.BusinessDate) {
			continue
		}
		if !matchIn(f.ProductIDs, r.ProductID) || !matchIn(f.WarehouseIDs, r.WarehouseID) {
			continue
		}
		if f.Series != "" && r.Series != f.Series {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return storageKey(out[i]) < storageKey(out[j]) })
	return out, nil
}

func (p planning) UpsertStorage(_ context.Context, rows []domain.DailyStoragePlan, actor string) (int, error) {
	p.s.lock()
	defer p.s.unlock()
	for _, r := range rows {
		k := storageKey(r)
		if prev, ok := p.s.d.storage[k]; ok {
			r.ID, r.CreatedAt, r.CreatedBy, r.RowVersion = prev.ID, prev.CreatedAt, prev.CreatedBy, prev.RowVersion
		} else {
			r.ID, r.CreatedAt, r.CreatedBy = uuid.NewString(), nowUTC(), actor
		}
		r.UpdatedAt, r.UpdatedBy, r.RowVersion = nowUTC(), actor, r.RowVersion+1
		p.s.d.storage[k] = r
	}
	return len(rows), nil
}

func (p planning) ListShipments(_ context.Context, f store.PlanFilter) ([]domain.DailyShipmentPlan, error) {
	p.s.lock()
	defer p.s.unlock()
	var out []domain.DailyShipmentPlan
	for _, r := range p.s.d.shipments {
		if !matchIn(f.VersionIDs, r.VersionID) || !inDateRange(f, r.BusinessDate) {
			continue
		}
		if !matchIn(f.ProductIDs, r.ProductID) || !matchIn(f.ChannelIDs, r.ChannelID) ||
			!matchIn(f.WarehouseIDs, r.WarehouseID) {
			continue
		}
		if f.Series != "" && r.Series != f.Series {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return shipmentKey(out[i]) < shipmentKey(out[j]) })
	return out, nil
}

func (p planning) UpsertShipments(_ context.Context, rows []domain.DailyShipmentPlan, actor string) (int, error) {
	p.s.lock()
	defer p.s.unlock()
	for _, r := range rows {
		k := shipmentKey(r)
		if prev, ok := p.s.d.shipments[k]; ok {
			r.ID, r.CreatedAt, r.CreatedBy, r.RowVersion = prev.ID, prev.CreatedAt, prev.CreatedBy, prev.RowVersion
		} else {
			r.ID, r.CreatedAt, r.CreatedBy = uuid.NewString(), nowUTC(), actor
		}
		r.UpdatedAt, r.UpdatedBy, r.RowVersion = nowUTC(), actor, r.RowVersion+1
		p.s.d.shipments[k] = r
	}
	return len(rows), nil
}

func (p planning) DeleteVersionRows(_ context.Context, versionID string) error {
	p.s.lock()
	defer p.s.unlock()
	for k, r := range p.s.d.cane {
		if r.VersionID == versionID {
			delete(p.s.d.cane, k)
		}
	}
	for k, r := range p.s.d.prodPlans {
		if r.VersionID == versionID {
			delete(p.s.d.prodPlans, k)
		}
	}
	for k, r := range p.s.d.storage {
		if r.VersionID == versionID {
			delete(p.s.d.storage, k)
		}
	}
	for k, r := range p.s.d.shipments {
		if r.VersionID == versionID {
			delete(p.s.d.shipments, k)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Downtime
// ---------------------------------------------------------------------------

func (p planning) ListDowntime(_ context.Context, f store.PlanFilter) ([]domain.DowntimeEvent, error) {
	p.s.lock()
	defer p.s.unlock()
	var out []domain.DowntimeEvent
	for _, e := range p.s.d.downtime {
		if !inDateRange(f, e.BusinessDate) {
			continue
		}
		if f.FactoryID != "" && e.FactoryID != f.FactoryID {
			continue
		}
		if !matchIn(f.LineIDs, e.LineID) {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BusinessDate < out[j].BusinessDate })
	return out, nil
}

func (p planning) SaveDowntime(_ context.Context, e domain.DowntimeEvent, actor string) (domain.DowntimeEvent, error) {
	p.s.lock()
	defer p.s.unlock()
	if e.ID == "" {
		e.ID = uuid.NewString()
		e.CreatedAt, e.CreatedBy = nowUTC(), actor
	} else if prev, ok := p.s.d.downtime[e.ID]; ok {
		if prev.RowVersion != e.RowVersion {
			return domain.DowntimeEvent{}, fmt.Errorf("%w: downtime event %s is at version %d",
				domain.ErrConflict, e.ID, prev.RowVersion)
		}
		e.CreatedAt, e.CreatedBy = prev.CreatedAt, prev.CreatedBy
	}
	e.UpdatedAt, e.UpdatedBy, e.RowVersion = nowUTC(), actor, e.RowVersion+1
	p.s.d.downtime[e.ID] = e
	return e, nil
}

func paginate[T any](items []T, opts store.ListOptions) store.Page[T] {
	total := len(items)
	if opts.Skip >= total {
		return store.Page[T]{Items: []T{}, Count: total}
	}
	end := opts.Skip + opts.Top
	if end > total {
		end = total
	}
	return store.Page[T]{Items: items[opts.Skip:end], Count: total}
}
