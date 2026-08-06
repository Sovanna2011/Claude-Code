// Package store defines the persistence boundary. The service layer depends on
// these interfaces only; the in-memory and PostgreSQL implementations live in
// sub-packages and are interchangeable.
package store

import (
	"context"

	"github.com/kss/sugarplan/internal/domain"
)

// ListOptions carries paging, sorting and the free-text search that every list
// endpoint supports. Filtering is deliberately not an open query language:
// each repository accepts only the filters it documents.
type ListOptions struct {
	Skip     int
	Top      int
	OrderBy  string
	Search   string
	Active   *bool
	ParentID string // company id, factory id, ... depending on the entity
}

// Normalise applies the API defaults and caps the page size so that a client
// cannot ask for an unbounded result set.
func (o ListOptions) Normalise() ListOptions {
	if o.Top <= 0 {
		o.Top = 100
	}
	if o.Top > 1000 {
		o.Top = 1000
	}
	if o.Skip < 0 {
		o.Skip = 0
	}
	return o
}

// Page is a slice of results plus the total count for the paging metadata.
type Page[T any] struct {
	Items []T `json:"value"`
	Count int `json:"count"`
}

// Repo is the uniform master-data repository. Every master entity supports the
// same five operations, so the service layer and the API handlers are written
// once and reused for all of them.
type Repo[T any] interface {
	List(ctx context.Context, opts ListOptions) (Page[T], error)
	Get(ctx context.Context, id string) (T, error)
	// GetByCode resolves the business key, which is what imports and the
	// SAPUI5 value helps use.
	GetByCode(ctx context.Context, code string) (T, error)
	// Save inserts when the id is empty and updates otherwise. An update whose
	// row version does not match the stored row returns domain.ErrConflict.
	Save(ctx context.Context, entity T, actor string) (T, error)
	// Deactivate is the soft delete: master data referenced by transactions is
	// never physically removed.
	Deactivate(ctx context.Context, id string, rowVersion int64, actor string) error
}

// MasterData groups the master-data repositories.
type MasterData interface {
	Companies() Repo[domain.Company]
	Factories() Repo[domain.Factory]
	Lines() Repo[domain.ProductionLine]
	Shifts() Repo[domain.Shift]
	ProductCategories() Repo[domain.ProductCategory]
	Products() Repo[domain.Product]
	UOMs() Repo[domain.UnitOfMeasure]
	UOMConversions() Repo[domain.UOMConversion]
	PackagingTypes() Repo[domain.PackagingType]
	Warehouses() Repo[domain.Warehouse]
	Customers() Repo[domain.Customer]
	Channels() Repo[domain.ShipmentChannel]
	Materials() Repo[domain.Material]
	ReasonCodes() Repo[domain.ReasonCode]
}

// PlanFilter selects daily planning rows. Empty fields mean "no restriction";
// every field is an allow-listed filter, never raw SQL from the client.
type PlanFilter struct {
	VersionIDs   []string
	FactoryID    string
	From         domain.BusinessDate
	To           domain.BusinessDate
	ProductIDs   []string
	WarehouseIDs []string
	ChannelIDs   []string
	LineIDs      []string
	Series       domain.Series
	Skip         int
	Top          int
}

// Planning is the repository for seasons, versions and the daily plan facts.
type Planning interface {
	// --- seasons and versions ---
	ListSeasons(ctx context.Context, opts ListOptions) (Page[domain.Season], error)
	GetSeason(ctx context.Context, id string) (domain.Season, error)
	SaveSeason(ctx context.Context, s domain.Season, actor string) (domain.Season, error)

	ListVersions(ctx context.Context, seasonID string, opts ListOptions) (Page[domain.PlanVersion], error)
	GetVersion(ctx context.Context, id string) (domain.PlanVersion, error)
	SaveVersion(ctx context.Context, v domain.PlanVersion, actor string) (domain.PlanVersion, error)

	ListAssumptions(ctx context.Context, versionID string) ([]domain.PlanAssumption, error)
	SaveAssumption(ctx context.Context, a domain.PlanAssumption, actor string) (domain.PlanAssumption, error)
	DeleteAssumption(ctx context.Context, id string) error

	ListMix(ctx context.Context, versionID string) ([]domain.ProductMixEntry, error)
	SaveMix(ctx context.Context, m domain.ProductMixEntry, actor string) (domain.ProductMixEntry, error)
	DeleteMix(ctx context.Context, id string) error

	// --- daily facts ---
	// The Upsert* methods match on the natural key of the row (version, date,
	// dimensions and series) so that re-running an import or re-generating a
	// plan replaces rows instead of duplicating them.
	ListCane(ctx context.Context, f PlanFilter) ([]domain.DailyCanePlan, error)
	UpsertCane(ctx context.Context, rows []domain.DailyCanePlan, actor string) (int, error)

	ListProducts(ctx context.Context, f PlanFilter) ([]domain.DailyProductPlan, error)
	UpsertProducts(ctx context.Context, rows []domain.DailyProductPlan, actor string) (int, error)

	ListStorage(ctx context.Context, f PlanFilter) ([]domain.DailyStoragePlan, error)
	UpsertStorage(ctx context.Context, rows []domain.DailyStoragePlan, actor string) (int, error)

	ListShipments(ctx context.Context, f PlanFilter) ([]domain.DailyShipmentPlan, error)
	UpsertShipments(ctx context.Context, rows []domain.DailyShipmentPlan, actor string) (int, error)

	// DeleteVersionRows clears every daily row of a version, used when a plan
	// is regenerated from its assumptions.
	DeleteVersionRows(ctx context.Context, versionID string) error

	// --- downtime ---
	ListDowntime(ctx context.Context, f PlanFilter) ([]domain.DowntimeEvent, error)
	SaveDowntime(ctx context.Context, e domain.DowntimeEvent, actor string) (domain.DowntimeEvent, error)
}

// ExecutionFilter selects execution records. Empty fields mean "no
// restriction"; every field is an allow-listed filter.
type ExecutionFilter struct {
	FactoryID   string
	VersionID   string
	From        domain.BusinessDate
	To          domain.BusinessDate
	ProductIDs  []string
	WarehouseID string
	LineID      string
	OrderID     string
	Statuses    []string
	// OpenOnly restricts to records that are still live: orders that are not
	// closed or cancelled, holds that are not released.
	OpenOnly bool
	Skip     int
	Top      int
}

// Document number series. Each one names the column its numbers live in, which
// is where the uniqueness that makes the numbering safe actually comes from.
const (
	SeriesOrder        = "PO" // production_orders.order_no
	SeriesConfirmation = "CF" // production_confirmations.confirmation_no
	SeriesDocument     = "MD" // inventory_documents.document_no
	SeriesSample       = "QS" // quality_samples.sample_no
)

// Execution is the repository for production orders, confirmations, inventory
// documents, stock balances, quality and maintenance.
type Execution interface {
	// --- production orders ---
	ListOrders(ctx context.Context, f ExecutionFilter) (Page[domain.ProductionOrder], error)
	GetOrder(ctx context.Context, id string) (domain.ProductionOrder, error)
	SaveOrder(ctx context.Context, o domain.ProductionOrder, actor string) (domain.ProductionOrder, error)

	// NextNumber issues the next number in a document series for a factory and
	// year, in the form SERIES-FACTORY-YEAR-NNNNN. The series is one of the
	// constants below; anything else is rejected rather than guessed at.
	//
	// The number is derived from the numbers that exist rather than from a
	// sequence, so a restored database does not start handing out numbers that
	// are already in use. Two callers racing for the same number is safe: the
	// unique constraint on the column refuses the loser, who asks again.
	NextNumber(ctx context.Context, series, factoryCode string, year int) (string, error)

	ListConfirmations(ctx context.Context, orderID string) ([]domain.ProductionConfirmation, error)
	GetConfirmation(ctx context.Context, id string) (domain.ProductionConfirmation, error)
	SaveConfirmation(ctx context.Context, c domain.ProductionConfirmation, actor string) (domain.ProductionConfirmation, error)

	// --- inventory ---
	// PostDocument writes the document, its items and the resulting balances.
	// The caller is responsible for having validated the posting; this method
	// is the write, not the decision.
	PostDocument(ctx context.Context, d domain.InventoryDocument, actor string) (domain.InventoryDocument, error)
	GetDocument(ctx context.Context, id string) (domain.InventoryDocument, error)
	ListDocuments(ctx context.Context, f ExecutionFilter) (Page[domain.InventoryDocument], error)
	// MarkReversed flags the original once its reversal is posted.
	MarkReversed(ctx context.Context, documentID string, actor string) error

	// Positions reads the current balances for the given warehouse/product
	// pairs, returning a zero position for a pair that has never been posted.
	Positions(ctx context.Context, pairs [][2]string) (map[string]domain.StockPosition, error)
	ListPositions(ctx context.Context, f ExecutionFilter) ([]domain.StockPosition, error)

	// --- quality ---
	ListParameters(ctx context.Context) ([]domain.QualityParameter, error)
	SaveParameter(ctx context.Context, p domain.QualityParameter, actor string) (domain.QualityParameter, error)
	// SpecsFor returns the specifications for a product effective on a date.
	SpecsFor(ctx context.Context, productID string, on domain.BusinessDate) ([]domain.QualitySpec, error)
	SaveSpec(ctx context.Context, s domain.QualitySpec, actor string) (domain.QualitySpec, error)

	ListSamples(ctx context.Context, f ExecutionFilter) (Page[domain.QualitySample], error)
	GetSample(ctx context.Context, id string) (domain.QualitySample, error)
	SaveSample(ctx context.Context, s domain.QualitySample, actor string) (domain.QualitySample, error)
	SaveResults(ctx context.Context, sampleID string, results []domain.QualityResult, actor string) error

	ListHolds(ctx context.Context, f ExecutionFilter) ([]domain.QualityHold, error)
	GetHold(ctx context.Context, id string) (domain.QualityHold, error)
	SaveHold(ctx context.Context, h domain.QualityHold, actor string) (domain.QualityHold, error)

	// --- maintenance ---
	ListMaintenance(ctx context.Context, f ExecutionFilter) ([]domain.MaintenanceWindow, error)
	SaveMaintenance(ctx context.Context, m domain.MaintenanceWindow, actor string) (domain.MaintenanceWindow, error)
}

// AuditFilter selects audit records.
type AuditFilter struct {
	Entity   string
	EntityID string
	Actor    string
	Action   string
	Skip     int
	Top      int
}

// Audit is the append-only audit trail. There is deliberately no update or
// delete method: application users cannot rewrite history.
type Audit interface {
	Append(ctx context.Context, e domain.AuditEvent) error
	List(ctx context.Context, f AuditFilter) (Page[domain.AuditEvent], error)
}

// Idempotency records the keys of posting requests that have already been
// processed, so a retried import or confirmation does not post twice.
type Idempotency interface {
	// Remember stores the key and returns false when it was already present,
	// together with the response body captured the first time.
	Remember(ctx context.Context, key, endpoint string, response []byte) (fresh bool, previous []byte, err error)
}

// Store is the whole persistence surface.
type Store interface {
	MasterData() MasterData
	Planning() Planning
	Execution() Execution
	Audit() Audit
	Idempotency() Idempotency
	// InTx runs fn inside a database transaction. Every posting that touches
	// more than one row - plan generation, release, bulk upsert - goes through
	// here so that a failure half way leaves nothing behind.
	InTx(ctx context.Context, fn func(Store) error) error
	Close() error
	// Ping backs the readiness probe.
	Ping(ctx context.Context) error
}
