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

type execution struct{ s *Store }

// Execution returns the execution repository.
func (s *Store) Execution() store.Execution { return execution{s} }

func positionKey(warehouseID, productID string) string { return warehouseID + "|" + productID }

func matchStatus(statuses []string, value string) bool {
	if len(statuses) == 0 {
		return true
	}
	for _, s := range statuses {
		if s == value {
			return true
		}
	}
	return false
}

func inRange(f store.ExecutionFilter, d domain.BusinessDate) bool {
	if f.From != "" && d < f.From {
		return false
	}
	if f.To != "" && d > f.To {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Production orders
// ---------------------------------------------------------------------------

func (e execution) ListOrders(_ context.Context, f store.ExecutionFilter) (store.Page[domain.ProductionOrder], error) {
	e.s.lock()
	defer e.s.unlock()

	var items []domain.ProductionOrder
	for _, o := range e.s.d.orders {
		if f.FactoryID != "" && o.FactoryID != f.FactoryID {
			continue
		}
		if f.VersionID != "" && o.VersionID != f.VersionID {
			continue
		}
		if f.LineID != "" && o.LineID != f.LineID {
			continue
		}
		if !matchIn(f.ProductIDs, o.ProductID) || !inRange(f, o.BusinessDate) {
			continue
		}
		if !matchStatus(f.Statuses, string(o.Status)) {
			continue
		}
		if f.OpenOnly && (o.Status == domain.OrderTechnicallyClosed ||
			o.Status == domain.OrderCancelled || o.Status == domain.OrderCompleted) {
			continue
		}
		items = append(items, o)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].OrderNo < items[j].OrderNo })
	return paginate(items, store.ListOptions{Skip: f.Skip, Top: orDefaultTop(f.Top)}), nil
}

func orDefaultTop(top int) int {
	if top <= 0 {
		return 500
	}
	return top
}

func (e execution) GetOrder(_ context.Context, id string) (domain.ProductionOrder, error) {
	e.s.lock()
	defer e.s.unlock()
	o, ok := e.s.d.orders[id]
	if !ok {
		return domain.ProductionOrder{}, fmt.Errorf("%w: production order %s", domain.ErrNotFound, id)
	}
	return o, nil
}

func (e execution) SaveOrder(_ context.Context, o domain.ProductionOrder, actor string) (domain.ProductionOrder, error) {
	e.s.lock()
	defer e.s.unlock()

	if o.Priority == 0 {
		o.Priority = 5 // the database default; the middle of the 1..9 band
	}
	if o.ID == "" {
		for _, existing := range e.s.d.orders {
			if existing.OrderNo == o.OrderNo {
				return domain.ProductionOrder{}, fmt.Errorf(
					"%w: production order %s already exists", domain.ErrDuplicate, o.OrderNo)
			}
		}
		o.ID = uuid.NewString()
		o.CreatedAt, o.CreatedBy = nowUTC(), actor
		o.UpdatedAt, o.UpdatedBy, o.RowVersion = o.CreatedAt, actor, 1
		e.s.d.orders[o.ID] = o
		return o, nil
	}
	prev, ok := e.s.d.orders[o.ID]
	if !ok {
		return domain.ProductionOrder{}, fmt.Errorf("%w: production order %s", domain.ErrNotFound, o.ID)
	}
	if prev.RowVersion != o.RowVersion {
		return domain.ProductionOrder{}, fmt.Errorf(
			"%w: production order %s was changed by %s (version %d, you have %d)",
			domain.ErrConflict, prev.OrderNo, prev.UpdatedBy, prev.RowVersion, o.RowVersion)
	}
	o.CreatedAt, o.CreatedBy = prev.CreatedAt, prev.CreatedBy
	o.UpdatedAt, o.UpdatedBy, o.RowVersion = nowUTC(), actor, prev.RowVersion+1
	e.s.d.orders[o.ID] = o
	return o, nil
}

func (e execution) NextOrderNo(_ context.Context, factoryCode string, year int) (string, error) {
	e.s.lock()
	defer e.s.unlock()
	prefix := fmt.Sprintf("PO-%s-%d-", factoryCode, year)
	max := 0
	for _, o := range e.s.d.orders {
		if !strings.HasPrefix(o.OrderNo, prefix) {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(strings.TrimPrefix(o.OrderNo, prefix), "%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("%s%05d", prefix, max+1), nil
}

// ---------------------------------------------------------------------------
// Confirmations
// ---------------------------------------------------------------------------

func (e execution) ListConfirmations(_ context.Context, orderID string) ([]domain.ProductionConfirmation, error) {
	e.s.lock()
	defer e.s.unlock()
	var out []domain.ProductionConfirmation
	for _, c := range e.s.d.confirmations {
		if orderID == "" || c.OrderID == orderID {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ConfirmationNo < out[j].ConfirmationNo })
	return out, nil
}

func (e execution) GetConfirmation(_ context.Context, id string) (domain.ProductionConfirmation, error) {
	e.s.lock()
	defer e.s.unlock()
	c, ok := e.s.d.confirmations[id]
	if !ok {
		return domain.ProductionConfirmation{}, fmt.Errorf("%w: confirmation %s", domain.ErrNotFound, id)
	}
	return c, nil
}

func (e execution) SaveConfirmation(_ context.Context, c domain.ProductionConfirmation, actor string) (domain.ProductionConfirmation, error) {
	e.s.lock()
	defer e.s.unlock()

	if c.ID == "" {
		for _, existing := range e.s.d.confirmations {
			if existing.ConfirmationNo == c.ConfirmationNo {
				return domain.ProductionConfirmation{}, fmt.Errorf(
					"%w: confirmation %s already exists", domain.ErrDuplicate, c.ConfirmationNo)
			}
		}
		c.ID = uuid.NewString()
		c.CreatedAt, c.CreatedBy = nowUTC(), actor
		c.UpdatedAt, c.UpdatedBy, c.RowVersion = c.CreatedAt, actor, 1
		for i := range c.Consumptions {
			c.Consumptions[i].ID = uuid.NewString()
			c.Consumptions[i].ConfirmationID = c.ID
			c.Consumptions[i].CreatedAt, c.Consumptions[i].CreatedBy = c.CreatedAt, actor
		}
		e.s.d.confirmations[c.ID] = c
		return c, nil
	}
	prev, ok := e.s.d.confirmations[c.ID]
	if !ok {
		return domain.ProductionConfirmation{}, fmt.Errorf("%w: confirmation %s", domain.ErrNotFound, c.ID)
	}
	c.CreatedAt, c.CreatedBy = prev.CreatedAt, prev.CreatedBy
	c.UpdatedAt, c.UpdatedBy, c.RowVersion = nowUTC(), actor, prev.RowVersion+1
	e.s.d.confirmations[c.ID] = c
	return c, nil
}

// ---------------------------------------------------------------------------
// Inventory
// ---------------------------------------------------------------------------

// PostDocument writes the document, its items and the resulting balances in one
// step. Callers run it inside InTx together with whatever else the posting
// implies, so a half-written posting cannot survive.
func (e execution) PostDocument(_ context.Context, d domain.InventoryDocument, actor string) (domain.InventoryDocument, error) {
	e.s.lock()
	defer e.s.unlock()

	for _, existing := range e.s.d.documents {
		if existing.DocumentNo == d.DocumentNo {
			return domain.InventoryDocument{}, fmt.Errorf(
				"%w: inventory document %s already exists", domain.ErrDuplicate, d.DocumentNo)
		}
	}

	d.ID = uuid.NewString()
	d.CreatedAt, d.CreatedBy = nowUTC(), actor
	if d.PostedAt.IsZero() {
		d.PostedAt = d.CreatedAt
	}
	for i := range d.Items {
		d.Items[i].ID = uuid.NewString()
		d.Items[i].DocumentID = d.ID
		d.Items[i].CreatedAt, d.Items[i].CreatedBy = d.CreatedAt, actor

		// Apply the line to the balance.
		key := positionKey(d.Items[i].WarehouseID, d.Items[i].ProductID)
		position, ok := e.s.d.positions[key]
		if !ok {
			position = domain.StockPosition{
				WarehouseID:  d.Items[i].WarehouseID,
				ProductID:    d.Items[i].ProductID,
				Quantity:     domain.Zero,
				HoldQuantity: domain.Zero,
			}
		}
		switch d.DocType {
		case domain.DocHold:
			position.HoldQuantity = position.HoldQuantity.Add(d.Items[i].Quantity)
		case domain.DocRelease:
			position.HoldQuantity = position.HoldQuantity.Sub(d.Items[i].Quantity)
		default:
			position.Quantity = position.Quantity.Add(d.Items[i].Quantity)
		}
		position.UpdatedAt, position.UpdatedBy = d.CreatedAt, actor
		position.RowVersion++
		e.s.d.positions[key] = position
	}

	e.s.d.documents[d.ID] = d
	return d, nil
}

func (e execution) GetDocument(_ context.Context, id string) (domain.InventoryDocument, error) {
	e.s.lock()
	defer e.s.unlock()
	d, ok := e.s.d.documents[id]
	if !ok {
		return domain.InventoryDocument{}, fmt.Errorf("%w: inventory document %s", domain.ErrNotFound, id)
	}
	return d, nil
}

func (e execution) ListDocuments(_ context.Context, f store.ExecutionFilter) (store.Page[domain.InventoryDocument], error) {
	e.s.lock()
	defer e.s.unlock()

	var items []domain.InventoryDocument
	for _, d := range e.s.d.documents {
		if f.FactoryID != "" && d.FactoryID != f.FactoryID {
			continue
		}
		if !inRange(f, d.BusinessDate) {
			continue
		}
		if f.WarehouseID != "" && !documentTouches(d, f.WarehouseID) {
			continue
		}
		items = append(items, d)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].BusinessDate != items[j].BusinessDate {
			return items[i].BusinessDate > items[j].BusinessDate // newest first
		}
		return items[i].DocumentNo < items[j].DocumentNo
	})
	return paginate(items, store.ListOptions{Skip: f.Skip, Top: orDefaultTop(f.Top)}), nil
}

func documentTouches(d domain.InventoryDocument, warehouseID string) bool {
	for _, item := range d.Items {
		if item.WarehouseID == warehouseID {
			return true
		}
	}
	return false
}

func (e execution) MarkReversed(_ context.Context, documentID, actor string) error {
	e.s.lock()
	defer e.s.unlock()
	d, ok := e.s.d.documents[documentID]
	if !ok {
		return fmt.Errorf("%w: inventory document %s", domain.ErrNotFound, documentID)
	}
	if d.Reversed {
		return fmt.Errorf("%w: document %s has already been reversed", domain.ErrValidation, d.DocumentNo)
	}
	d.Reversed = true
	e.s.d.documents[documentID] = d
	return nil
}

func (e execution) Positions(_ context.Context, pairs [][2]string) (map[string]domain.StockPosition, error) {
	e.s.lock()
	defer e.s.unlock()

	out := map[string]domain.StockPosition{}
	for _, p := range pairs {
		key := positionKey(p[0], p[1])
		if position, ok := e.s.d.positions[key]; ok {
			out[key] = position
			continue
		}
		// A pair that has never been posted has a zero position, not a missing
		// one: the first receipt into a new store must not be a special case.
		out[key] = domain.StockPosition{
			WarehouseID: p[0], ProductID: p[1],
			Quantity: domain.Zero, HoldQuantity: domain.Zero,
		}
	}
	return out, nil
}

func (e execution) ListPositions(_ context.Context, f store.ExecutionFilter) ([]domain.StockPosition, error) {
	e.s.lock()
	defer e.s.unlock()

	var out []domain.StockPosition
	for _, p := range e.s.d.positions {
		if f.WarehouseID != "" && p.WarehouseID != f.WarehouseID {
			continue
		}
		if !matchIn(f.ProductIDs, p.ProductID) {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return positionKey(out[i].WarehouseID, out[i].ProductID) <
			positionKey(out[j].WarehouseID, out[j].ProductID)
	})
	return out, nil
}

// ---------------------------------------------------------------------------
// Quality
// ---------------------------------------------------------------------------

func (e execution) ListParameters(_ context.Context) ([]domain.QualityParameter, error) {
	e.s.lock()
	defer e.s.unlock()
	var out []domain.QualityParameter
	for _, p := range e.s.d.qualityParams {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func (e execution) SaveParameter(_ context.Context, p domain.QualityParameter, actor string) (domain.QualityParameter, error) {
	e.s.lock()
	defer e.s.unlock()
	// The code is the business key, so saving a parameter that already exists
	// updates it rather than failing: the catalogue is configuration, and a
	// re-run of a seed or an import must be idempotent.
	if p.ID == "" {
		for _, existing := range e.s.d.qualityParams {
			if existing.Code == p.Code {
				p.ID, p.RowVersion = existing.ID, existing.RowVersion
				p.CreatedAt, p.CreatedBy = existing.CreatedAt, existing.CreatedBy
				break
			}
		}
	}
	if p.ID == "" {
		p.ID = uuid.NewString()
		p.CreatedAt, p.CreatedBy = nowUTC(), actor
		p.RowVersion = 0
	}
	p.UpdatedAt, p.UpdatedBy, p.RowVersion = nowUTC(), actor, p.RowVersion+1
	e.s.d.qualityParams[p.ID] = p
	return p, nil
}

func (e execution) SpecsFor(_ context.Context, productID string, on domain.BusinessDate) ([]domain.QualitySpec, error) {
	e.s.lock()
	defer e.s.unlock()
	var out []domain.QualitySpec
	for _, s := range e.s.d.qualitySpecs {
		if s.ProductID != productID || !s.IsEffectiveOn(on) {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ParameterID < out[j].ParameterID })
	return out, nil
}

func (e execution) SaveSpec(_ context.Context, s domain.QualitySpec, actor string) (domain.QualitySpec, error) {
	e.s.lock()
	defer e.s.unlock()
	if s.ID == "" {
		for _, existing := range e.s.d.qualitySpecs {
			if existing.ProductID == s.ProductID && existing.ParameterID == s.ParameterID &&
				existing.ValidFrom == s.ValidFrom {
				s.ID, s.RowVersion = existing.ID, existing.RowVersion
				s.CreatedAt, s.CreatedBy = existing.CreatedAt, existing.CreatedBy
				break
			}
		}
	}
	if s.ID == "" {
		s.ID = uuid.NewString()
		s.CreatedAt, s.CreatedBy = nowUTC(), actor
		s.RowVersion = 0
	}
	s.UpdatedAt, s.UpdatedBy, s.RowVersion = nowUTC(), actor, s.RowVersion+1
	e.s.d.qualitySpecs[s.ID] = s
	return s, nil
}

func (e execution) ListSamples(_ context.Context, f store.ExecutionFilter) (store.Page[domain.QualitySample], error) {
	e.s.lock()
	defer e.s.unlock()

	var items []domain.QualitySample
	for _, s := range e.s.d.samples {
		if f.FactoryID != "" && s.FactoryID != f.FactoryID {
			continue
		}
		if !matchIn(f.ProductIDs, s.ProductID) || !inRange(f, s.BusinessDate) {
			continue
		}
		s.Results = append([]domain.QualityResult(nil), e.s.d.results[s.ID]...)
		items = append(items, s)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SampleNo > items[j].SampleNo })
	return paginate(items, store.ListOptions{Skip: f.Skip, Top: orDefaultTop(f.Top)}), nil
}

func (e execution) GetSample(_ context.Context, id string) (domain.QualitySample, error) {
	e.s.lock()
	defer e.s.unlock()
	s, ok := e.s.d.samples[id]
	if !ok {
		return domain.QualitySample{}, fmt.Errorf("%w: quality sample %s", domain.ErrNotFound, id)
	}
	s.Results = append([]domain.QualityResult(nil), e.s.d.results[id]...)
	return s, nil
}

func (e execution) SaveSample(_ context.Context, s domain.QualitySample, actor string) (domain.QualitySample, error) {
	e.s.lock()
	defer e.s.unlock()
	if s.Status == "" {
		s.Status = "OPEN"
	}
	if s.TakenAt.IsZero() {
		s.TakenAt = nowUTC()
	}
	if s.ID == "" {
		for _, existing := range e.s.d.samples {
			if existing.SampleNo == s.SampleNo {
				return domain.QualitySample{}, fmt.Errorf(
					"%w: sample %s already exists", domain.ErrDuplicate, s.SampleNo)
			}
		}
		s.ID = uuid.NewString()
		s.CreatedAt, s.CreatedBy = nowUTC(), actor
		s.RowVersion = 0
	}
	s.UpdatedAt, s.UpdatedBy, s.RowVersion = nowUTC(), actor, s.RowVersion+1
	stored := s
	stored.Results = nil // results live in their own map
	e.s.d.samples[s.ID] = stored
	return s, nil
}

func (e execution) SaveResults(_ context.Context, sampleID string, results []domain.QualityResult, actor string) error {
	e.s.lock()
	defer e.s.unlock()
	if _, ok := e.s.d.samples[sampleID]; !ok {
		return fmt.Errorf("%w: quality sample %s", domain.ErrNotFound, sampleID)
	}
	stored := make([]domain.QualityResult, 0, len(results))
	for _, r := range results {
		if r.ID == "" {
			r.ID = uuid.NewString()
		}
		r.SampleID = sampleID
		r.CreatedAt, r.CreatedBy = nowUTC(), actor
		stored = append(stored, r)
	}
	e.s.d.results[sampleID] = stored
	return nil
}

func (e execution) ListHolds(_ context.Context, f store.ExecutionFilter) ([]domain.QualityHold, error) {
	e.s.lock()
	defer e.s.unlock()
	var out []domain.QualityHold
	for _, h := range e.s.d.holds {
		if f.WarehouseID != "" && h.WarehouseID != f.WarehouseID {
			continue
		}
		if !matchIn(f.ProductIDs, h.ProductID) {
			continue
		}
		if f.OpenOnly && !h.IsOpen() {
			continue
		}
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlacedOn > out[j].PlacedOn })
	return out, nil
}

func (e execution) GetHold(_ context.Context, id string) (domain.QualityHold, error) {
	e.s.lock()
	defer e.s.unlock()
	h, ok := e.s.d.holds[id]
	if !ok {
		return domain.QualityHold{}, fmt.Errorf("%w: quality hold %s", domain.ErrNotFound, id)
	}
	return h, nil
}

func (e execution) SaveHold(_ context.Context, h domain.QualityHold, actor string) (domain.QualityHold, error) {
	e.s.lock()
	defer e.s.unlock()
	if h.ID == "" {
		h.ID = uuid.NewString()
		h.CreatedAt, h.CreatedBy = nowUTC(), actor
		h.RowVersion = 0
	} else if prev, ok := e.s.d.holds[h.ID]; ok {
		if prev.RowVersion != h.RowVersion {
			return domain.QualityHold{}, fmt.Errorf(
				"%w: quality hold %s is at version %d", domain.ErrConflict, h.ID, prev.RowVersion)
		}
		h.CreatedAt, h.CreatedBy = prev.CreatedAt, prev.CreatedBy
	}
	h.UpdatedAt, h.UpdatedBy, h.RowVersion = nowUTC(), actor, h.RowVersion+1
	e.s.d.holds[h.ID] = h
	return h, nil
}

// ---------------------------------------------------------------------------
// Maintenance
// ---------------------------------------------------------------------------

func (e execution) ListMaintenance(_ context.Context, f store.ExecutionFilter) ([]domain.MaintenanceWindow, error) {
	e.s.lock()
	defer e.s.unlock()
	var out []domain.MaintenanceWindow
	for _, m := range e.s.d.maintenance {
		if f.FactoryID != "" && m.FactoryID != f.FactoryID {
			continue
		}
		if !matchStatus(f.Statuses, m.Status) {
			continue
		}
		// A window overlaps the range if it starts before the end and ends
		// after the start.
		if f.To != "" && m.StartDate > f.To {
			continue
		}
		if f.From != "" && m.EndDate < f.From {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartDate < out[j].StartDate })
	return out, nil
}

func (e execution) SaveMaintenance(_ context.Context, m domain.MaintenanceWindow, actor string) (domain.MaintenanceWindow, error) {
	e.s.lock()
	defer e.s.unlock()
	if m.Status == "" {
		m.Status = domain.MaintenancePlanned
	}
	if m.ID == "" {
		m.ID = uuid.NewString()
		m.CreatedAt, m.CreatedBy = nowUTC(), actor
		m.RowVersion = 0
	} else if prev, ok := e.s.d.maintenance[m.ID]; ok {
		if prev.RowVersion != m.RowVersion {
			return domain.MaintenanceWindow{}, fmt.Errorf(
				"%w: maintenance window %s is at version %d",
				domain.ErrConflict, m.ID, prev.RowVersion)
		}
		m.CreatedAt, m.CreatedBy = prev.CreatedAt, prev.CreatedBy
	}
	m.UpdatedAt, m.UpdatedBy, m.RowVersion = nowUTC(), actor, m.RowVersion+1
	e.s.d.maintenance[m.ID] = m
	return m, nil
}
