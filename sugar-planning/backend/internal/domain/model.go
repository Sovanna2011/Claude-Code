package domain

import "time"

// BusinessDate is a calendar day in the factory time zone. It is stored as an
// ISO date (yyyy-mm-dd) and never carries a time component, because a
// production day is a business concept, not an instant.
type BusinessDate string

// NewBusinessDate formats a time as a business date.
func NewBusinessDate(t time.Time) BusinessDate { return BusinessDate(t.Format("2006-01-02")) }

// Time parses the business date back into a UTC midnight time value.
func (b BusinessDate) Time() (time.Time, error) { return time.Parse("2006-01-02", string(b)) }

// MustTime parses the business date, returning the zero time on failure.
func (b BusinessDate) MustTime() time.Time {
	t, err := b.Time()
	if err != nil {
		return time.Time{}
	}
	return t
}

// AddDays returns the business date shifted by n days.
func (b BusinessDate) AddDays(n int) BusinessDate {
	return NewBusinessDate(b.MustTime().AddDate(0, 0, n))
}

// Before reports whether b is earlier than other.
func (b BusinessDate) Before(other BusinessDate) bool { return b < other }

// DaysBetween returns the number of days from b to other (other - b).
func (b BusinessDate) DaysBetween(other BusinessDate) int {
	return int(other.MustTime().Sub(b.MustTime()).Hours() / 24)
}

// Valid reports whether the value parses as an ISO date.
func (b BusinessDate) Valid() bool { _, err := b.Time(); return err == nil }

// ---------------------------------------------------------------------------
// Enumerations
// ---------------------------------------------------------------------------

// PlanType classifies a planning version (section 4).
type PlanType string

const (
	PlanTypeBudget         PlanType = "BUDGET"          // original season plan
	PlanTypeForecast       PlanType = "FORECAST"        // rolling forecast
	PlanTypeRevised        PlanType = "REVISED"         // revised plan
	PlanTypeWhatIf         PlanType = "WHATIF"          // simulation, never released
	PlanTypeBaseline       PlanType = "BASELINE"        // approved baseline
	PlanTypeActual         PlanType = "ACTUAL"          // recorded actuals
	PlanTypeLatestEstimate PlanType = "LATEST_ESTIMATE" // actual to date + forecast ahead
)

// ValidPlanType reports whether the value is a known plan type.
func ValidPlanType(p PlanType) bool {
	switch p {
	case PlanTypeBudget, PlanTypeForecast, PlanTypeRevised, PlanTypeWhatIf,
		PlanTypeBaseline, PlanTypeActual, PlanTypeLatestEstimate:
		return true
	}
	return false
}

// PlanStatus is the workflow status of a planning version (section 4).
type PlanStatus string

const (
	StatusDraft      PlanStatus = "DRAFT"
	StatusInReview   PlanStatus = "IN_REVIEW"
	StatusApproved   PlanStatus = "APPROVED"
	StatusReleased   PlanStatus = "RELEASED"
	StatusSuperseded PlanStatus = "SUPERSEDED"
	StatusRejected   PlanStatus = "REJECTED"
	StatusClosed     PlanStatus = "CLOSED"
)

// Series separates planned values from recorded actuals. An actual posting can
// never overwrite an approved plan because the two live in different series
// and, in practice, in different versions.
type Series string

const (
	SeriesPlan   Series = "PLAN"
	SeriesActual Series = "ACTUAL"
)

// ProcessStage identifies where in the factory flow a measure belongs.
type ProcessStage string

const (
	StageCane     ProcessStage = "CANE"
	StageRawSugar ProcessStage = "RAW_SUGAR"
	StageRemelt   ProcessStage = "REMELT"
	StageRefining ProcessStage = "REFINING"
	StagePacking  ProcessStage = "PACKING"
	StageShipment ProcessStage = "SHIPMENT"
)

// StorageClass separates raw sugar storage from finished goods storage, since
// they have independent capacity pools.
type StorageClass string

const (
	StorageRaw      StorageClass = "RAW"
	StorageFinished StorageClass = "FINISHED"
	StorageMaterial StorageClass = "MATERIAL"
)

// OrderStatus is the production order life cycle (section 10).
type OrderStatus string

const (
	OrderPlanned            OrderStatus = "PLANNED"
	OrderReleased           OrderStatus = "RELEASED"
	OrderInProcess          OrderStatus = "IN_PROCESS"
	OrderPartiallyConfirmed OrderStatus = "PARTIALLY_CONFIRMED"
	OrderCompleted          OrderStatus = "COMPLETED"
	OrderTechnicallyClosed  OrderStatus = "TECHNICALLY_CLOSED"
	OrderCancelled          OrderStatus = "CANCELLED"
)

// QualityStatus is the verdict of a quality result against its specification.
type QualityStatus string

const (
	QualityPass    QualityStatus = "PASS"
	QualityWarning QualityStatus = "WARNING"
	QualityFail    QualityStatus = "FAIL"
)

// Severity classifies alerts and exceptions using the Fiori semantic colours.
type Severity string

const (
	SeverityInfo    Severity = "INFO"
	SeverityWarning Severity = "WARNING"
	SeverityError   Severity = "ERROR"
	SeveritySuccess Severity = "SUCCESS"
)

// ---------------------------------------------------------------------------
// Audit and concurrency
// ---------------------------------------------------------------------------

// AuditFields are embedded in every mutable entity.
type AuditFields struct {
	CreatedAt  time.Time `json:"createdAt"`
	CreatedBy  string    `json:"createdBy"`
	UpdatedAt  time.Time `json:"updatedAt"`
	UpdatedBy  string    `json:"updatedBy"`
	RowVersion int64     `json:"rowVersion"`
}

// Version exposes the optimistic-concurrency row version. Every entity that
// embeds AuditFields therefore satisfies the small interface the HTTP layer
// uses to emit an ETag.
func (a AuditFields) Version() int64 { return a.RowVersion }

// Validity is the effective-dating common to all master data.
type Validity struct {
	ValidFrom BusinessDate `json:"validFrom"`
	ValidTo   BusinessDate `json:"validTo"`
	Active    bool         `json:"active"`
}

// IsEffective reports whether the record is active on the given date.
func (v Validity) IsEffective(on BusinessDate) bool {
	if !v.Active {
		return false
	}
	if v.ValidFrom != "" && on < v.ValidFrom {
		return false
	}
	if v.ValidTo != "" && on > v.ValidTo {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Organisation and master data
// ---------------------------------------------------------------------------

// Company is the top level of the organisational hierarchy.
type Company struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	TimeZone string `json:"timeZone"`
	Validity
	AuditFields
}

// Factory is a physical plant belonging to a company.
type Factory struct {
	ID        string `json:"id"`
	CompanyID string `json:"companyId"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	TimeZone  string `json:"timeZone"`
	Validity
	AuditFields
}

// ProductionLine is a crushing, refining or packing line with a rated capacity.
type ProductionLine struct {
	ID        string       `json:"id"`
	FactoryID string       `json:"factoryId"`
	Code      string       `json:"code"`
	Name      string       `json:"name"`
	Stage     ProcessStage `json:"stage"`
	RatedTPH  Dec          `json:"ratedTph"` // rated throughput, tons per hour
	Validity
	AuditFields
}

// Shift is a named working period inside a business day.
type Shift struct {
	ID        string `json:"id"`
	FactoryID string `json:"factoryId"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	StartTime string `json:"startTime"` // HH:MM in factory local time
	Hours     Dec    `json:"hours"`
	Validity
	AuditFields
}

// ProductCategory groups products for reporting and mix analysis.
type ProductCategory struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Validity
	AuditFields
}

// Product is anything planned, produced, stored or shipped.
type Product struct {
	ID           string       `json:"id"`
	Code         string       `json:"code"`
	Name         string       `json:"name"`
	CategoryCode string       `json:"categoryCode"`
	BaseUOM      string       `json:"baseUom"`
	Stage        ProcessStage `json:"stage"`
	StorageClass StorageClass `json:"storageClass"`
	IsFinished   bool         `json:"isFinished"`
	Validity
	AuditFields
}

// UnitOfMeasure is a measurement unit with its display precision.
type UnitOfMeasure struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Dimension string `json:"dimension"` // MASS, COUNT, VOLUME, TIME
	Decimals  int    `json:"decimals"`
	Validity
	AuditFields
}

// UOMConversion converts a quantity from one unit to another for a product, or
// for every product when ProductID is empty.
type UOMConversion struct {
	ID        string `json:"id"`
	ProductID string `json:"productId,omitempty"`
	FromUOM   string `json:"fromUom"`
	ToUOM     string `json:"toUom"`
	Factor    Dec    `json:"factor"` // quantity_to = quantity_from * factor
	Validity
	AuditFields
}

// PackagingType describes how a product is packed, with the net weight that
// drives packaging material requirements.
type PackagingType struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	NetWeightKg Dec    `json:"netWeightKg"`
	MaterialID  string `json:"materialId,omitempty"` // primary packaging material
	Validity
	AuditFields
}

// PackagingBOMLine is one component consumed per package of a packaging type.
//
// The primary bag is not held here - a packaging type points at its own bag
// material - because a bag is one per package by definition and giving it a
// quantity would invite somebody to set it to two. This is for everything else:
// the liner inside the jumbo bag, the thread that sews it, the label on it.
type PackagingBOMLine struct {
	ID          string `json:"id"`
	PackagingID string `json:"packagingId"`
	MaterialID  string `json:"materialId"`
	// QtyPerPackage is in the material's own unit, at scale 6: a thread
	// consumption of 0.0035 spools per bag is a real figure, not a rounding
	// error.
	QtyPerPackage Dec `json:"qtyPerPackage"`
	Validity
	AuditFields
}

// Warehouse is a storage location with a nominal and a usable capacity.
type Warehouse struct {
	ID             string       `json:"id"`
	FactoryID      string       `json:"factoryId"`
	Code           string       `json:"code"`
	Name           string       `json:"name"`
	StorageClass   StorageClass `json:"storageClass"`
	CapacityTons   Dec          `json:"capacityTons"`
	UsablePct      Dec          `json:"usablePct"` // usable share of nominal capacity
	OpeningBalance Dec          `json:"openingBalance"`
	IsSilo         bool         `json:"isSilo"`
	Validity
	AuditFields
}

// UsableCapacity is the capacity that may actually be filled.
func (w Warehouse) UsableCapacity() Dec {
	return RoundQty(w.CapacityTons.Mul(w.UsablePct).Div(DI(100)))
}

// Customer is a sold-to party.
type Customer struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Country string `json:"country"`
	Validity
	AuditFields
}

// ShipmentChannel is a sales or dispatch channel. The reference workbook tracks
// shipment per trader; those become configurable channels here rather than
// hard-coded columns.
type ShipmentChannel struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	CustomerID string `json:"customerId,omitempty"`
	Category   string `json:"category"` // QUOTA, EXPORT, DOMESTIC, INTERNAL
	Validity
	AuditFields
}

// Material is a consumable or packaging item subject to requirement planning.
type Material struct {
	ID           string `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	UOM          string `json:"uom"`
	SafetyStock  Dec    `json:"safetyStock"`
	LeadTimeDays int    `json:"leadTimeDays"`
	ScrapPct     Dec    `json:"scrapPct"`
	OnHand       Dec    `json:"onHand"`
	OnOrder      Dec    `json:"onOrder"`
	Validity
	AuditFields
}

// ReasonCode explains a variance, loss, adjustment, downtime or rework.
type ReasonCode struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Category string `json:"category"` // VARIANCE, LOSS, ADJUSTMENT, DOWNTIME, REJECTION, REWORK, STOCK_CORRECTION
	Validity
	AuditFields
}

// ---------------------------------------------------------------------------
// Season and planning versions
// ---------------------------------------------------------------------------

// Season is a crushing campaign, for example 2026-2027.
type Season struct {
	ID          string       `json:"id"`
	CompanyID   string       `json:"companyId"`
	FactoryID   string       `json:"factoryId"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	StartDate   BusinessDate `json:"startDate"`
	EndDate     BusinessDate `json:"endDate"`
	PlannedDays int          `json:"plannedDays"`
	Status      string       `json:"status"` // OPEN, CLOSED
	AuditFields
}

// PlanVersion is one scenario of one season (section 4).
type PlanVersion struct {
	ID            string       `json:"id"`
	SeasonID      string       `json:"seasonId"`
	VersionNo     int          `json:"versionNo"`
	Code          string       `json:"code"`
	Description   string       `json:"description"`
	PlanType      PlanType     `json:"planType"`
	Status        PlanStatus   `json:"status"`
	Owner         string       `json:"owner"`
	SourceVersion string       `json:"sourceVersionId,omitempty"`
	EffectiveFrom BusinessDate `json:"effectiveFrom"`
	EffectiveTo   BusinessDate `json:"effectiveTo"`
	SubmittedAt   *time.Time   `json:"submittedAt,omitempty"`
	ApprovedAt    *time.Time   `json:"approvedAt,omitempty"`
	ApprovedBy    string       `json:"approvedBy,omitempty"`
	ReleasedAt    *time.Time   `json:"releasedAt,omitempty"`
	LockedThrough BusinessDate `json:"lockedThrough,omitempty"`
	Comment       string       `json:"comment,omitempty"`
	AuditFields
}

// IsEditable reports whether ordinary editing of plan rows is allowed.
func (v PlanVersion) IsEditable() bool {
	return v.Status == StatusDraft || v.Status == StatusRejected
}

// PlanAssumption is an effective-dated, named input to the calculations. Every
// rate, capacity and factor used by the generator is stored here so that no
// business number is hard-coded in the code.
type PlanAssumption struct {
	ID          string       `json:"id"`
	VersionID   string       `json:"versionId"`
	Code        string       `json:"code"`
	Description string       `json:"description"`
	Value       Dec          `json:"value"`
	UOM         string       `json:"uom"`
	ValidFrom   BusinessDate `json:"validFrom"`
	ValidTo     BusinessDate `json:"validTo"`
	AuditFields
}

// Well-known assumption codes. Sites may add their own; the generator only
// requires the ones listed here and reports a validation error when missing.
const (
	AsmCaneTarget = "CANE_TARGET_TONS"
	AsmSeasonDays = "SEASON_DAYS"
	// AsmCampaignDays is the whole planning horizon, which is longer than the
	// crushing season: the mill goes on refining stored raw sugar, and selling,
	// for months after the last cane is crushed. Absent, the plan covers the
	// crushing season only.
	AsmCampaignDays       = "CAMPAIGN_DAYS"
	AsmRecoveryPct        = "RAW_RECOVERY_PCT"
	AsmDirectToRefinePct  = "RAW_DIRECT_TO_REFINE_PCT"
	AsmRemeltInputFactor  = "REMELT_INPUT_FACTOR"
	AsmCrushRateTPH       = "CRUSH_RATE_TPH"
	AsmAvailableHours     = "AVAILABLE_HOURS_PER_DAY"
	AsmQuotaShipmentTPD   = "QUOTA_SHIPMENT_TPD"
	AsmJumboPackTPD       = "JUMBO_PACK_TPD"
	AsmJumboBagWeightTons = "JUMBO_BAG_WEIGHT_TONS"
	AsmCapacityWarnPct    = "CAPACITY_WARN_PCT"
	AsmCapacityAlertPct   = "CAPACITY_ALERT_PCT"
	AsmMassBalanceTolPct  = "MASS_BALANCE_TOLERANCE_PCT"
	AsmRecoveryMinPct     = "RECOVERY_MIN_PCT"
	AsmRecoveryMaxPct     = "RECOVERY_MAX_PCT"
	// AsmSupplyTolerancePct is how far the committed cane may sit from the
	// season target before the supply plan says so. Two per cent by default:
	// a season is contracted months ahead and nobody expects it to land on the
	// tonne, but a tenth out is a different plan.
	AsmSupplyTolerancePct = "SUPPLY_TOLERANCE_PCT"
	// The wash-out cadence and the rate the mill winds down to the day before
	// one. Scalars, so they are assumptions; the ramp, the run-down and any
	// adjusted wash-out dates are ordered and live in plan_crushing_steps.
	AsmCleaningEveryDays = "CLEANING_EVERY_DAYS"
	AsmPreCleaningRate   = "PRE_CLEANING_RATE_TONS"
)

// ProductMixEntry is the share of finished goods output planned for one
// product/packaging combination. Shares are expressed in tons for the season,
// which is how the reference workbook states them.
type ProductMixEntry struct {
	ID          string `json:"id"`
	VersionID   string `json:"versionId"`
	ProductID   string `json:"productId"`
	PackagingID string `json:"packagingId,omitempty"`
	SeasonTons  Dec    `json:"seasonTons"`
	WarehouseID string `json:"warehouseId,omitempty"`
	// DailyRateTons caps the daily output for this entry. Zero means "spread
	// the season tonnage evenly over the campaign". A positive value makes the
	// generator run the entry at that rate until the season tonnage is used up,
	// which is how jumbo bag packing is planned (300 t/day up to 20,700 t).
	DailyRateTons Dec    `json:"dailyRateTons"`
	LineID        string `json:"lineId,omitempty"`
	AuditFields
}

// ---------------------------------------------------------------------------
// Daily planning facts
// ---------------------------------------------------------------------------

// DailyCanePlan is one day (optionally one shift) of cane supply and crushing.
type DailyCanePlan struct {
	ID            string       `json:"id"`
	VersionID     string       `json:"versionId"`
	FactoryID     string       `json:"factoryId"`
	BusinessDate  BusinessDate `json:"businessDate"`
	ShiftID       string       `json:"shiftId,omitempty"`
	Series        Series       `json:"series"`
	CaneAvailable Dec          `json:"caneAvailable"`
	CaneDelivered Dec          `json:"caneDelivered"`
	CaneAccepted  Dec          `json:"caneAccepted"`
	CaneRejected  Dec          `json:"caneRejected"`
	CaneDiverted  Dec          `json:"caneDiverted"`
	CaneCrushed   Dec          `json:"caneCrushed"`
	CrushRateTPH  Dec          `json:"crushRateTph"`
	AvailableHrs  Dec          `json:"availableHours"`
	StoppageHrs   Dec          `json:"stoppageHours"`
	ReasonCode    string       `json:"reasonCode,omitempty"`
	Note          string       `json:"note,omitempty"`
	AuditFields
}

// RunHours is available hours minus stoppage hours, floored at zero.
func (d DailyCanePlan) RunHours() Dec {
	return ClampNonNegative(d.AvailableHrs.Sub(d.StoppageHrs))
}

// DailyProductPlan is one day of production for a product/packaging on a line.
type DailyProductPlan struct {
	ID           string       `json:"id"`
	VersionID    string       `json:"versionId"`
	FactoryID    string       `json:"factoryId"`
	LineID       string       `json:"lineId,omitempty"`
	BusinessDate BusinessDate `json:"businessDate"`
	ShiftID      string       `json:"shiftId,omitempty"`
	ProductID    string       `json:"productId"`
	PackagingID  string       `json:"packagingId,omitempty"`
	Series       Series       `json:"series"`
	Quantity     Dec          `json:"quantity"`    // good output in tons
	RemeltInput  Dec          `json:"remeltInput"` // raw sugar consumed
	ProcessLoss  Dec          `json:"processLoss"` // loss in tons
	Rework       Dec          `json:"rework"`      // reworked quantity
	Rejected     Dec          `json:"rejected"`    // rejected quantity
	HoldQty      Dec          `json:"holdQty"`     // quality hold
	ReasonCode   string       `json:"reasonCode,omitempty"`
	Note         string       `json:"note,omitempty"`
	AuditFields
}

// DailyStoragePlan is one day of the stock ledger for a product in a warehouse
// or silo (section 9).
type DailyStoragePlan struct {
	ID                string       `json:"id"`
	VersionID         string       `json:"versionId"`
	WarehouseID       string       `json:"warehouseId"`
	ProductID         string       `json:"productId"`
	BusinessDate      BusinessDate `json:"businessDate"`
	Series            Series       `json:"series"`
	BeginningBalance  Dec          `json:"beginningBalance"`
	ProductionReceipt Dec          `json:"productionReceipt"`
	TransferIn        Dec          `json:"transferIn"`
	TransferOut       Dec          `json:"transferOut"`
	RepackIn          Dec          `json:"repackIn"`
	RepackOut         Dec          `json:"repackOut"`
	RemeltIssue       Dec          `json:"remeltIssue"`
	ShipmentQty       Dec          `json:"shipmentQty"`
	Adjustment        Dec          `json:"adjustment"`
	ProcessLoss       Dec          `json:"processLoss"`
	HoldQty           Dec          `json:"holdQty"`
	EndingBalance     Dec          `json:"endingBalance"`
	PhysicalBalance   *Dec         `json:"physicalBalance,omitempty"`
	AuditFields
}

// DailyShipmentPlan is one day of shipment for a product and channel.
type DailyShipmentPlan struct {
	ID           string       `json:"id"`
	VersionID    string       `json:"versionId"`
	WarehouseID  string       `json:"warehouseId,omitempty"`
	ProductID    string       `json:"productId"`
	ChannelID    string       `json:"channelId"`
	BusinessDate BusinessDate `json:"businessDate"`
	Series       Series       `json:"series"`
	Quantity     Dec          `json:"quantity"`
	Note         string       `json:"note,omitempty"`
	AuditFields
}

// ---------------------------------------------------------------------------
// Execution, quality and downtime (data model for phase 4)
// ---------------------------------------------------------------------------

// DowntimeEvent records planned or unplanned lost production time.
type DowntimeEvent struct {
	ID           string       `json:"id"`
	FactoryID    string       `json:"factoryId"`
	LineID       string       `json:"lineId,omitempty"`
	BusinessDate BusinessDate `json:"businessDate"`
	ShiftID      string       `json:"shiftId,omitempty"`
	StartAt      time.Time    `json:"startAt"`
	EndAt        time.Time    `json:"endAt"`
	DurationHrs  Dec          `json:"durationHours"`
	Planned      bool         `json:"planned"`
	ReasonCode   string       `json:"reasonCode"`
	RootCause    string       `json:"rootCause,omitempty"`
	Team         string       `json:"team,omitempty"`
	Action       string       `json:"correctiveAction,omitempty"`
	AuditFields
}

// AuditEvent is the append-only audit trail record (section 19).
type AuditEvent struct {
	ID            string    `json:"id"`
	OccurredAt    time.Time `json:"occurredAt"`
	Actor         string    `json:"actor"`
	Action        string    `json:"action"`
	Entity        string    `json:"entity"`
	EntityID      string    `json:"entityId"`
	Before        string    `json:"before,omitempty"`
	After         string    `json:"after,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	CorrelationID string    `json:"correlationId"`
	SourceIP      string    `json:"sourceIp,omitempty"`
}

// Notification is an alert that has been sent to somebody.
//
// The distinction from Alert matters. An alert is calculated: it is true of the
// plan at the moment somebody looks. A notification is a fact about a person -
// this was raised, they were told, they have or have not read it - and it
// survives the alert ceasing to be true, which is what makes an inbox an
// account of what happened rather than a second dashboard.
type Notification struct {
	ID string `json:"id"`
	// Recipient is a **role code**, not a person. There is no user directory
	// here - identity, roles and data scope all come from the token - so a
	// notification is addressed to "the shipment planners at this factory" and
	// somebody's inbox is what their roles and their scope entitle them to see.
	// It is also the better answer operationally: an alert addressed to a person
	// who has left is an alert nobody owns.
	Recipient string     `json:"recipient"`
	FactoryID string     `json:"factoryId,omitempty"`
	Severity  Severity   `json:"severity"`
	Code      string     `json:"code"`
	Title     string     `json:"title"`
	Detail    string     `json:"detail,omitempty"`
	Entity    string     `json:"entity,omitempty"`
	EntityID  string     `json:"entityId,omitempty"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// IsRead reports whether the recipient has seen it.
func (n Notification) IsRead() bool { return n.ReadAt != nil }

// DedupeKey is what stops the same alert being raised at every tick.
//
// It is the recipient, the code and the thing it is about - not the wording,
// which may be regenerated with a different figure in it, and not the date it
// was raised, which would make every day's tick a new notification.
func (n Notification) DedupeKey() string {
	return n.Recipient + "|" + n.FactoryID + "|" + n.Code + "|" + n.Entity + "|" + n.EntityID
}

// Alert is a calculated exception surfaced on the dashboard and in the inbox.
type Alert struct {
	Code     string       `json:"code"`
	Severity Severity     `json:"severity"`
	Title    string       `json:"title"`
	Detail   string       `json:"detail"`
	Entity   string       `json:"entity,omitempty"`
	EntityID string       `json:"entityId,omitempty"`
	Date     BusinessDate `json:"date,omitempty"`
}
