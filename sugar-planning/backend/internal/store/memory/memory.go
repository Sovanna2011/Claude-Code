// Package memory is an in-memory implementation of the store interfaces.
//
// It is not a test double with hard-coded answers: it enforces the same
// business keys, optimistic concurrency and transaction rollback semantics as
// the PostgreSQL implementation, which is what makes it usable both for the
// unit tests and for the `demo` run profile that needs no database.
package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// data is the whole database. It is cloned to take a transaction snapshot.
type data struct {
	companies  map[string]domain.Company
	factories  map[string]domain.Factory
	lines      map[string]domain.ProductionLine
	shifts     map[string]domain.Shift
	categories map[string]domain.ProductCategory
	products   map[string]domain.Product
	uoms       map[string]domain.UnitOfMeasure
	uomConv    map[string]domain.UOMConversion
	packaging  map[string]domain.PackagingType
	warehouses map[string]domain.Warehouse
	customers  map[string]domain.Customer
	channels   map[string]domain.ShipmentChannel
	materials  map[string]domain.Material
	reasons    map[string]domain.ReasonCode
	caneSrc    map[string]domain.CaneSource

	seasons     map[string]domain.Season
	versions    map[string]domain.PlanVersion
	assumptions map[string]domain.PlanAssumption
	mix         map[string]domain.ProductMixEntry
	supply      map[string]domain.CaneSupplyEntry

	cane      map[string]domain.DailyCanePlan
	caneSup   map[string]domain.DailyCaneSupply
	prodPlans map[string]domain.DailyProductPlan
	storage   map[string]domain.DailyStoragePlan
	shipments map[string]domain.DailyShipmentPlan
	downtime  map[string]domain.DowntimeEvent

	orders        map[string]domain.ProductionOrder
	confirmations map[string]domain.ProductionConfirmation
	documents     map[string]domain.InventoryDocument
	positions     map[string]domain.StockPosition
	qualityParams map[string]domain.QualityParameter
	qualitySpecs  map[string]domain.QualitySpec
	samples       map[string]domain.QualitySample
	results       map[string][]domain.QualityResult
	holds         map[string]domain.QualityHold
	maintenance   map[string]domain.MaintenanceWindow

	costElements  map[string]domain.CostElement
	costRates     map[string]domain.CostRate
	exchangeRates map[string]domain.ExchangeRate
	costRuns      map[string]domain.CostRun

	importMappings map[string]domain.ImportMapping
	importJobs     map[string]domain.ImportJob
	importRows     map[string][]domain.ImportRow

	notifications map[string]domain.Notification
	savedViews    map[string]domain.SavedView

	batches      map[string]domain.Batch
	packagingBOM map[string]domain.PackagingBOMLine

	outbox map[string]domain.OutboxEvent
	// jobs holds what each scheduled job last did; jobLeases holds when the
	// current holder's claim runs out, which is not part of the record an
	// operator reads and so is kept apart from it.
	jobs      map[string]domain.JobRun
	jobLeases map[string]time.Time

	audit []domain.AuditEvent
	idem  map[string][]byte
}

func newData() *data {
	return &data{
		companies: map[string]domain.Company{}, factories: map[string]domain.Factory{},
		lines: map[string]domain.ProductionLine{}, shifts: map[string]domain.Shift{},
		categories: map[string]domain.ProductCategory{}, products: map[string]domain.Product{},
		uoms: map[string]domain.UnitOfMeasure{}, uomConv: map[string]domain.UOMConversion{},
		packaging: map[string]domain.PackagingType{}, warehouses: map[string]domain.Warehouse{},
		customers: map[string]domain.Customer{}, channels: map[string]domain.ShipmentChannel{},
		materials: map[string]domain.Material{}, reasons: map[string]domain.ReasonCode{},
		seasons: map[string]domain.Season{}, versions: map[string]domain.PlanVersion{},
		assumptions: map[string]domain.PlanAssumption{}, mix: map[string]domain.ProductMixEntry{},
		caneSrc: map[string]domain.CaneSource{}, supply: map[string]domain.CaneSupplyEntry{},
		caneSup: map[string]domain.DailyCaneSupply{},
		cane:    map[string]domain.DailyCanePlan{}, prodPlans: map[string]domain.DailyProductPlan{},
		storage: map[string]domain.DailyStoragePlan{}, shipments: map[string]domain.DailyShipmentPlan{},
		downtime:       map[string]domain.DowntimeEvent{},
		batches:        map[string]domain.Batch{},
		notifications:  map[string]domain.Notification{},
		savedViews:     map[string]domain.SavedView{},
		importMappings: map[string]domain.ImportMapping{},
		importJobs:     map[string]domain.ImportJob{},
		importRows:     map[string][]domain.ImportRow{},
		packagingBOM:   map[string]domain.PackagingBOMLine{},
		outbox:         map[string]domain.OutboxEvent{},
		jobs:           map[string]domain.JobRun{},
		jobLeases:      map[string]time.Time{},
		costElements:   map[string]domain.CostElement{},
		costRates:      map[string]domain.CostRate{},
		exchangeRates:  map[string]domain.ExchangeRate{},
		costRuns:       map[string]domain.CostRun{},
		orders:         map[string]domain.ProductionOrder{},
		confirmations:  map[string]domain.ProductionConfirmation{},
		documents:      map[string]domain.InventoryDocument{},
		positions:      map[string]domain.StockPosition{},
		qualityParams:  map[string]domain.QualityParameter{},
		qualitySpecs:   map[string]domain.QualitySpec{},
		samples:        map[string]domain.QualitySample{},
		results:        map[string][]domain.QualityResult{},
		holds:          map[string]domain.QualityHold{},
		maintenance:    map[string]domain.MaintenanceWindow{},
		idem:           map[string][]byte{},
	}
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// clone takes a snapshot for transaction rollback. Entity values are structs
// and decimal values are immutable, so a shallow copy of each map is a correct
// deep copy of the state.
func (d *data) clone() *data {
	return &data{
		companies: cloneMap(d.companies), factories: cloneMap(d.factories),
		lines: cloneMap(d.lines), shifts: cloneMap(d.shifts),
		categories: cloneMap(d.categories), products: cloneMap(d.products),
		uoms: cloneMap(d.uoms), uomConv: cloneMap(d.uomConv),
		packaging: cloneMap(d.packaging), warehouses: cloneMap(d.warehouses),
		customers: cloneMap(d.customers), channels: cloneMap(d.channels),
		materials: cloneMap(d.materials), reasons: cloneMap(d.reasons),
		seasons: cloneMap(d.seasons), versions: cloneMap(d.versions),
		assumptions: cloneMap(d.assumptions), mix: cloneMap(d.mix),
		caneSrc: cloneMap(d.caneSrc), supply: cloneMap(d.supply), caneSup: cloneMap(d.caneSup),
		cane: cloneMap(d.cane), prodPlans: cloneMap(d.prodPlans),
		storage: cloneMap(d.storage), shipments: cloneMap(d.shipments),
		downtime: cloneMap(d.downtime),
		orders:   cloneMap(d.orders), confirmations: cloneMap(d.confirmations),
		documents: cloneDocuments(d.documents), positions: cloneMap(d.positions),
		qualityParams: cloneMap(d.qualityParams), qualitySpecs: cloneMap(d.qualitySpecs),
		samples: cloneMap(d.samples), results: cloneResults(d.results),
		holds: cloneMap(d.holds), maintenance: cloneMap(d.maintenance),
		costElements: cloneMap(d.costElements), costRates: cloneMap(d.costRates),
		exchangeRates: cloneMap(d.exchangeRates), costRuns: cloneRuns(d.costRuns),
		batches:        cloneMap(d.batches),
		notifications:  cloneMap(d.notifications),
		importMappings: cloneMap(d.importMappings),
		importJobs:     cloneMap(d.importJobs),
		importRows:     cloneRows(d.importRows),
		packagingBOM:   cloneMap(d.packagingBOM),
		outbox:         cloneMap(d.outbox),
		jobs:           cloneMap(d.jobs),
		jobLeases:      cloneMap(d.jobLeases),
		audit:          append([]domain.AuditEvent(nil), d.audit...),
		idem:           cloneMap(d.idem),
	}
}

// cloneDocuments deep-copies the documents, whose item slices would otherwise
// be shared with the snapshot and survive a rollback.
func cloneDocuments(m map[string]domain.InventoryDocument) map[string]domain.InventoryDocument {
	out := make(map[string]domain.InventoryDocument, len(m))
	for k, v := range m {
		v.Items = append([]domain.InventoryDocumentItem(nil), v.Items...)
		out[k] = v
	}
	return out
}

// cloneRuns deep-copies the saved cost runs, whose line slices would otherwise
// be shared with the snapshot and survive a rollback.
func cloneRuns(m map[string]domain.CostRun) map[string]domain.CostRun {
	out := make(map[string]domain.CostRun, len(m))
	for k, v := range m {
		v.Lines = append([]domain.CostLine(nil), v.Lines...)
		out[k] = v
	}
	return out
}

// cloneResults deep-copies the per-sample result slices, for the same reason.
func cloneResults(m map[string][]domain.QualityResult) map[string][]domain.QualityResult {
	out := make(map[string][]domain.QualityResult, len(m))
	for k, v := range m {
		out[k] = append([]domain.QualityResult(nil), v...)
	}
	return out
}

// cloneRows deep-copies the staged import rows, so a rolled-back commit leaves
// the staging area as it was.
func cloneRows(m map[string][]domain.ImportRow) map[string][]domain.ImportRow {
	out := make(map[string][]domain.ImportRow, len(m))
	for k, v := range m {
		out[k] = append([]domain.ImportRow(nil), v...)
	}
	return out
}

// locker lets a transaction child share the parent's state without deadlocking
// on the parent's mutex.
type locker interface {
	Lock()
	Unlock()
}

type nopLock struct{}

func (nopLock) Lock()   {}
func (nopLock) Unlock() {}

// Store is the in-memory store.
type Store struct {
	mu     locker
	parent *Store // set on a transaction child
	d      *data
}

// New creates an empty store.
func New() *Store {
	return &Store{mu: &sync.Mutex{}, d: newData()}
}

func nowUTC() time.Time { return time.Now().UTC() }

func (s *Store) lock()   { s.mu.Lock() }
func (s *Store) unlock() { s.mu.Unlock() }

// Close releases nothing; it exists to satisfy the interface.
func (s *Store) Close() error { return nil }

// Ping always succeeds for the in-memory store.
func (s *Store) Ping(context.Context) error { return nil }

// InTx snapshots the state, runs fn against a child store that shares the same
// data, and restores the snapshot if fn fails. That gives the same
// all-or-nothing guarantee as a database transaction, which is what the
// posting tests assert.
func (s *Store) InTx(ctx context.Context, fn func(store.Store) error) error {
	if s.parent != nil {
		// Nested call inside an existing transaction: join it.
		return fn(s)
	}
	s.lock()
	defer s.unlock()

	snapshot := s.d.clone()
	child := &Store{mu: nopLock{}, parent: s, d: s.d}
	if err := fn(child); err != nil {
		s.d = snapshot
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Generic master-data repository
// ---------------------------------------------------------------------------

// spec adapts an entity type to the generic repository: it says how to read and
// write the fields the repository needs, without the repository knowing
// anything else about the entity.
type spec[T any] struct {
	name   string
	id     func(*T) *string
	code   func(T) string
	audit  func(*T) *domain.AuditFields
	active func(*T) *bool
	parent func(T) string
	text   func(T) string
}

type memRepo[T any] struct {
	s   *Store
	sel func(*data) map[string]T
	sp  spec[T]
}

func (r memRepo[T]) List(_ context.Context, opts store.ListOptions) (store.Page[T], error) {
	opts = opts.Normalise()
	r.s.lock()
	defer r.s.unlock()

	var items []T
	for _, v := range r.sel(r.s.d) {
		e := v
		if opts.Active != nil && r.sp.active != nil && *r.sp.active(&e) != *opts.Active {
			continue
		}
		if opts.ParentID != "" && r.sp.parent != nil && r.sp.parent(e) != opts.ParentID {
			continue
		}
		if opts.Search != "" && r.sp.text != nil &&
			!strings.Contains(strings.ToLower(r.sp.text(e)), strings.ToLower(opts.Search)) {
			continue
		}
		items = append(items, e)
	}
	sort.Slice(items, func(a, b int) bool { return r.sp.code(items[a]) < r.sp.code(items[b]) })

	total := len(items)
	if opts.Skip >= total {
		return store.Page[T]{Items: []T{}, Count: total}, nil
	}
	end := opts.Skip + opts.Top
	if end > total {
		end = total
	}
	return store.Page[T]{Items: items[opts.Skip:end], Count: total}, nil
}

func (r memRepo[T]) Get(_ context.Context, id string) (T, error) {
	r.s.lock()
	defer r.s.unlock()
	v, ok := r.sel(r.s.d)[id]
	if !ok {
		var zero T
		return zero, fmt.Errorf("%w: %s %s", domain.ErrNotFound, r.sp.name, id)
	}
	return v, nil
}

func (r memRepo[T]) GetByCode(_ context.Context, code string) (T, error) {
	r.s.lock()
	defer r.s.unlock()
	for _, v := range r.sel(r.s.d) {
		if r.sp.code(v) == code {
			return v, nil
		}
	}
	var zero T
	return zero, fmt.Errorf("%w: %s with code %s", domain.ErrNotFound, r.sp.name, code)
}

func (r memRepo[T]) Save(_ context.Context, entity T, actor string) (T, error) {
	r.s.lock()
	defer r.s.unlock()

	m := r.sel(r.s.d)
	idPtr := r.sp.id(&entity)
	aud := r.sp.audit(&entity)
	var zero T

	if *idPtr == "" {
		for _, v := range m {
			if r.sp.code(v) == r.sp.code(entity) {
				return zero, fmt.Errorf("%w: %s %s already exists", domain.ErrDuplicate, r.sp.name, r.sp.code(entity))
			}
		}
		*idPtr = uuid.NewString()
		aud.CreatedAt, aud.CreatedBy = nowUTC(), actor
		aud.UpdatedAt, aud.UpdatedBy = aud.CreatedAt, actor
		aud.RowVersion = 1
		m[*idPtr] = entity
		return entity, nil
	}

	existing, ok := m[*idPtr]
	if !ok {
		return zero, fmt.Errorf("%w: %s %s", domain.ErrNotFound, r.sp.name, *idPtr)
	}
	prev := r.sp.audit(&existing)
	if aud.RowVersion != prev.RowVersion {
		return zero, fmt.Errorf("%w: %s %s was changed by %s (version %d, you have %d)",
			domain.ErrConflict, r.sp.name, *idPtr, prev.UpdatedBy, prev.RowVersion, aud.RowVersion)
	}
	for id, v := range m {
		if id != *idPtr && r.sp.code(v) == r.sp.code(entity) {
			return zero, fmt.Errorf("%w: %s %s already exists", domain.ErrDuplicate, r.sp.name, r.sp.code(entity))
		}
	}
	aud.CreatedAt, aud.CreatedBy = prev.CreatedAt, prev.CreatedBy
	aud.UpdatedAt, aud.UpdatedBy = nowUTC(), actor
	aud.RowVersion = prev.RowVersion + 1
	m[*idPtr] = entity
	return entity, nil
}

func (r memRepo[T]) Deactivate(_ context.Context, id string, rowVersion int64, actor string) error {
	r.s.lock()
	defer r.s.unlock()

	m := r.sel(r.s.d)
	existing, ok := m[id]
	if !ok {
		return fmt.Errorf("%w: %s %s", domain.ErrNotFound, r.sp.name, id)
	}
	aud := r.sp.audit(&existing)
	if aud.RowVersion != rowVersion {
		return fmt.Errorf("%w: %s %s is at version %d", domain.ErrConflict, r.sp.name, id, aud.RowVersion)
	}
	if r.sp.active == nil {
		return fmt.Errorf("%w: %s cannot be deactivated", domain.ErrValidation, r.sp.name)
	}
	*r.sp.active(&existing) = false
	aud.UpdatedAt, aud.UpdatedBy = nowUTC(), actor
	aud.RowVersion++
	m[id] = existing
	return nil
}
