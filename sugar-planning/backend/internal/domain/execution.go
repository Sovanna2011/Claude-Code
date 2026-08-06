package domain

import (
	"fmt"
	"time"
)

// This file holds the execution side of the model: inventory documents,
// production orders and confirmations, and quality. As everywhere else in this
// package, the types and rules here perform no I/O; the service layer reads the
// current position, asks these functions whether a posting is allowed, and
// writes the result in one transaction.

// ---------------------------------------------------------------------------
// Inventory documents
// ---------------------------------------------------------------------------

// DocType classifies an inventory document.
type DocType string

const (
	DocReceipt    DocType = "RECEIPT"
	DocIssue      DocType = "ISSUE"
	DocTransfer   DocType = "TRANSFER"
	DocAdjustment DocType = "ADJUSTMENT"
	DocCount      DocType = "COUNT"
	DocHold       DocType = "HOLD"
	DocRelease    DocType = "RELEASE"
	DocShipment   DocType = "SHIPMENT"
	DocReversal   DocType = "REVERSAL"
)

// ValidDocType reports whether the value is a known document type.
func ValidDocType(t DocType) bool {
	switch t {
	case DocReceipt, DocIssue, DocTransfer, DocAdjustment, DocCount,
		DocHold, DocRelease, DocShipment, DocReversal:
		return true
	}
	return false
}

// InventoryDocument is one posting. A balance is never edited directly: it is
// the consequence of documents, and a mistake is corrected by a reversal that
// points back at the original.
type InventoryDocument struct {
	ID           string                  `json:"id"`
	DocumentNo   string                  `json:"documentNo"`
	DocType      DocType                 `json:"docType"`
	BusinessDate BusinessDate            `json:"businessDate"`
	PostedAt     time.Time               `json:"postedAt"`
	FactoryID    string                  `json:"factoryId"`
	Reference    string                  `json:"reference,omitempty"`
	ReversalOf   string                  `json:"reversalOf,omitempty"`
	Reversed     bool                    `json:"reversed"`
	ReasonCode   string                  `json:"reasonCode,omitempty"`
	Note         string                  `json:"note,omitempty"`
	Items        []InventoryDocumentItem `json:"items"`
	CreatedAt    time.Time               `json:"createdAt"`
	CreatedBy    string                  `json:"createdBy"`
}

// InventoryDocumentItem is one line of a posting.
//
// Quantity is signed: a receipt is positive, an issue negative. Keeping the
// sign on the line is what makes a reversal trivially correct - it is the same
// lines with the sign flipped - and what lets one document mix directions, as a
// transfer does.
type InventoryDocumentItem struct {
	ID          string `json:"id"`
	DocumentID  string `json:"documentId"`
	LineNo      int    `json:"lineNo"`
	WarehouseID string `json:"warehouseId"`
	ProductID   string `json:"productId"`
	BatchID     string `json:"batchId,omitempty"`
	Quantity    Dec    `json:"quantity"`
	UOM         string `json:"uom"`
	// ToWarehouse is set on a transfer line, which the service expands into a
	// negative line on the source and a positive line on the target.
	ToWarehouse string    `json:"toWarehouse,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	CreatedBy   string    `json:"createdBy"`
}

// StockPosition is the current balance of one product in one warehouse.
type StockPosition struct {
	WarehouseID  string    `json:"warehouseId"`
	ProductID    string    `json:"productId"`
	Quantity     Dec       `json:"quantity"`
	HoldQuantity Dec       `json:"holdQuantity"`
	UpdatedAt    time.Time `json:"updatedAt"`
	UpdatedBy    string    `json:"updatedBy"`
	RowVersion   int64     `json:"rowVersion"`
}

// Available is the quantity that may actually be issued: what is there, less
// what quality has blocked.
func (s StockPosition) Available() Dec {
	return RoundQty(s.Quantity.Sub(s.HoldQuantity))
}

// PostingOptions carry the authorisations that let a posting break a rule.
// Both are refused unless the caller holds the permission and the service
// records the override on the audit event.
type PostingOptions struct {
	// AllowNegativeStock permits a balance to go below zero. Sites that post
	// consumption before receipt sometimes need it; it is off by default.
	AllowNegativeStock bool
	// CapacityOverride permits a receipt that takes a store past its usable
	// capacity.
	CapacityOverride bool
}

// PostingLine is one line of a posting checked against the current position.
type PostingLine struct {
	LineNo    int
	Warehouse Warehouse
	Position  StockPosition
	// Delta is the signed change this line makes to the quantity.
	Delta Dec
	// HoldDelta is the signed change to the held quantity, used by hold and
	// release documents.
	HoldDelta Dec
}

// CheckPosting applies the inventory rules from sections 9 and 20 to a whole
// document and reports every problem at once, so a rejected posting names all
// its offending lines rather than one at a time.
//
// The rules, in the order a warehouse keeper would think of them:
//
//   - a posting may not drive stock below zero;
//   - quality-held stock may be neither shipped nor consumed;
//   - a receipt may not take a store past its usable capacity;
//   - a release may not free more than is held.
func CheckPosting(lines []PostingLine, opts PostingOptions) error {
	verr := &ValidationError{}

	for _, line := range lines {
		resulting := line.Position.Quantity.Add(line.Delta)
		resultingHold := line.Position.HoldQuantity.Add(line.HoldDelta)

		if resulting.IsNegative() && !opts.AllowNegativeStock {
			verr.AddRow(line.LineNo, "quantity", "NEGATIVE_STOCK", fmt.Sprintf(
				"%s holds %s t of this product; issuing %s t would leave %s t",
				warehouseLabel(line.Warehouse), RoundQty(line.Position.Quantity),
				RoundQty(line.Delta.Abs()), RoundQty(resulting)))
			continue
		}

		// An issue draws on the available balance, not the total: held stock is
		// physically present but blocked.
		if line.Delta.IsNegative() {
			available := line.Position.Available()
			if line.Delta.Abs().GreaterThan(available) && !opts.AllowNegativeStock {
				verr.AddRow(line.LineNo, "quantity", "QUALITY_HELD", fmt.Sprintf(
					"%s has %s t available; %s t of the %s t on hand is on quality hold",
					warehouseLabel(line.Warehouse), available,
					RoundQty(line.Position.HoldQuantity), RoundQty(line.Position.Quantity)))
				continue
			}
		}

		if line.Delta.GreaterThan(Zero) {
			usable := line.Warehouse.UsableCapacity()
			if usable.GreaterThan(Zero) && resulting.GreaterThan(usable) && !opts.CapacityOverride {
				verr.AddRow(line.LineNo, "quantity", "CAPACITY_EXCEEDED", fmt.Sprintf(
					"%s has %s t of usable capacity; this receipt would take it to %s t. "+
						"An authorised override is required to store beyond capacity",
					warehouseLabel(line.Warehouse), usable, RoundQty(resulting)))
				continue
			}
		}

		if resultingHold.IsNegative() {
			verr.AddRow(line.LineNo, "holdQuantity", "HOLD_UNDERFLOW", fmt.Sprintf(
				"%s has %s t on hold; releasing %s t is more than is held",
				warehouseLabel(line.Warehouse), RoundQty(line.Position.HoldQuantity),
				RoundQty(line.HoldDelta.Abs())))
			continue
		}
		if resultingHold.GreaterThan(resulting) {
			verr.AddRow(line.LineNo, "holdQuantity", "HOLD_EXCEEDS_STOCK", fmt.Sprintf(
				"%s cannot hold %s t when the balance would be %s t",
				warehouseLabel(line.Warehouse), RoundQty(resultingHold), RoundQty(resulting)))
		}
	}

	return verr.OrNil()
}

func warehouseLabel(w Warehouse) string {
	if w.Name != "" {
		return w.Code + " " + w.Name
	}
	if w.Code != "" {
		return w.Code
	}
	return "the warehouse"
}

// ReverseDocument builds the reversal of a posted document: the same lines with
// the sign flipped, pointing back at the original.
//
// Nothing is deleted and nothing is edited. Both documents stay in the ledger,
// which is what lets somebody six months later see that a mistake was made and
// what was done about it.
func ReverseDocument(original InventoryDocument, businessDate BusinessDate, reason, actor string) (InventoryDocument, error) {
	if original.Reversed {
		return InventoryDocument{}, fmt.Errorf(
			"%w: document %s has already been reversed", ErrValidation, original.DocumentNo)
	}
	if original.DocType == DocReversal {
		return InventoryDocument{}, fmt.Errorf(
			"%w: %s is itself a reversal; reverse the original instead",
			ErrValidation, original.DocumentNo)
	}
	if reason == "" {
		return InventoryDocument{}, fmt.Errorf(
			"%w: a reversal requires a reason for the audit trail", ErrValidation)
	}
	if businessDate == "" {
		businessDate = original.BusinessDate
	}

	reversal := InventoryDocument{
		DocType:      DocReversal,
		BusinessDate: businessDate,
		FactoryID:    original.FactoryID,
		Reference:    original.DocumentNo,
		ReversalOf:   original.ID,
		ReasonCode:   original.ReasonCode,
		Note:         reason,
		CreatedBy:    actor,
	}
	for _, item := range original.Items {
		reversal.Items = append(reversal.Items, InventoryDocumentItem{
			LineNo:      item.LineNo,
			WarehouseID: item.WarehouseID,
			ProductID:   item.ProductID,
			BatchID:     item.BatchID,
			Quantity:    item.Quantity.Neg(),
			UOM:         item.UOM,
			CreatedBy:   actor,
		})
	}
	return reversal, nil
}

// ---------------------------------------------------------------------------
// Production orders
// ---------------------------------------------------------------------------

// ProductionOrder is a released instruction to produce a quantity of a product
// on a line, on a date.
type ProductionOrder struct {
	ID             string       `json:"id"`
	OrderNo        string       `json:"orderNo"`
	CompanyID      string       `json:"companyId"`
	FactoryID      string       `json:"factoryId"`
	LineID         string       `json:"lineId,omitempty"`
	VersionID      string       `json:"versionId,omitempty"`
	BusinessDate   BusinessDate `json:"businessDate"`
	ShiftID        string       `json:"shiftId,omitempty"`
	ProductID      string       `json:"productId"`
	PackagingID    string       `json:"packagingId,omitempty"`
	PlannedQty     Dec          `json:"plannedQty"`
	ConfirmedQty   Dec          `json:"confirmedQty"`
	BatchID        string       `json:"batchId,omitempty"`
	BOMVersion     string       `json:"bomVersion,omitempty"`
	PlannedStart   *time.Time   `json:"plannedStart,omitempty"`
	PlannedEnd     *time.Time   `json:"plannedEnd,omitempty"`
	Priority       int          `json:"priority"`
	Status         OrderStatus  `json:"status"`
	Team           string       `json:"team,omitempty"`
	VarianceReason string       `json:"varianceReason,omitempty"`
	AuditFields
}

// OpenQty is what is still to be confirmed.
func (o ProductionOrder) OpenQty() Dec {
	return RoundQty(ClampNonNegative(o.PlannedQty.Sub(o.ConfirmedQty)))
}

// VariancePct is how far the confirmed quantity is from the plan.
func (o ProductionOrder) VariancePct() Dec {
	return RoundPct(SafePct(o.ConfirmedQty.Sub(o.PlannedQty), o.PlannedQty))
}

// OrderAction is a command applied to a production order.
type OrderAction string

const (
	OrderActionRelease  OrderAction = "RELEASE"
	OrderActionConfirm  OrderAction = "CONFIRM"
	OrderActionComplete OrderAction = "COMPLETE"
	OrderActionClose    OrderAction = "TECHNICALLY_CLOSE"
	OrderActionCancel   OrderAction = "CANCEL"
)

// orderTransitions is the order life cycle from section 10. Confirmation is
// not in this table because where it lands depends on the quantity, which
// ConfirmedStatus decides.
var orderTransitions = []struct {
	Action     OrderAction
	From       OrderStatus
	To         OrderStatus
	Permission string
}{
	{OrderActionRelease, OrderPlanned, OrderReleased, PermActualProduce},
	{OrderActionCancel, OrderPlanned, OrderCancelled, PermActualProduce},
	{OrderActionCancel, OrderReleased, OrderCancelled, PermActualProduce},
	{OrderActionComplete, OrderPartiallyConfirmed, OrderCompleted, PermActualProduce},
	{OrderActionComplete, OrderInProcess, OrderCompleted, PermActualProduce},
	{OrderActionClose, OrderCompleted, OrderTechnicallyClosed, PermActualProduce},
	{OrderActionClose, OrderPartiallyConfirmed, OrderTechnicallyClosed, PermActualProduce},
}

// ApplyOrderAction validates an order command and returns the resulting status.
func ApplyOrderAction(o ProductionOrder, action OrderAction) (OrderStatus, string, error) {
	for _, t := range orderTransitions {
		if t.From == o.Status && t.Action == action {
			return t.To, t.Permission, nil
		}
	}
	return "", "", fmt.Errorf("%w: cannot %s order %s in status %s",
		ErrStateTransition, action, o.OrderNo, o.Status)
}

// AllowedOrderActions lists what may be done to an order in its current status.
func AllowedOrderActions(o ProductionOrder) []OrderAction {
	var out []OrderAction
	for _, t := range orderTransitions {
		if t.From == o.Status {
			out = append(out, t.Action)
		}
	}
	if o.Status == OrderReleased || o.Status == OrderInProcess ||
		o.Status == OrderPartiallyConfirmed {
		out = append(out, OrderActionConfirm)
	}
	return out
}

// ConfirmedStatus works out where an order lands after a confirmation.
//
// An order that has met its planned quantity is complete; one that has been
// confirmed but not fully is partially confirmed. Over-confirmation is allowed
// - factories produce more than planned - and still completes the order.
func ConfirmedStatus(planned, confirmed Dec) OrderStatus {
	switch {
	case confirmed.LessThanOrEqual(Zero):
		return OrderReleased
	case confirmed.GreaterThanOrEqual(planned):
		return OrderCompleted
	default:
		return OrderPartiallyConfirmed
	}
}

// CheckOrderClose applies the rule from section 10: an order closes only after
// quantity reconciliation, or with an authorised variance reason.
func CheckOrderClose(o ProductionOrder, tolerancePct Dec, varianceReason string, authorised bool) error {
	if o.PlannedQty.LessThanOrEqual(Zero) {
		return nil
	}
	difference := o.ConfirmedQty.Sub(o.PlannedQty).Abs()
	limit := o.PlannedQty.Mul(tolerancePct).Div(DI(100))
	if difference.LessThanOrEqual(limit) {
		return nil
	}
	if varianceReason == "" {
		return fmt.Errorf(
			"%w: order %s confirmed %s t against a plan of %s t (%s%%), which is outside the %s%% tolerance; "+
				"closing it needs a variance reason",
			ErrValidation, o.OrderNo, RoundQty(o.ConfirmedQty), RoundQty(o.PlannedQty),
			o.VariancePct(), tolerancePct)
	}
	if !authorised {
		return fmt.Errorf(
			"%w: closing order %s with a variance of %s%% needs an authorised approver",
			ErrForbidden, o.OrderNo, o.VariancePct())
	}
	return nil
}

// ProductionConfirmation records what a shift actually produced.
type ProductionConfirmation struct {
	ID             string                `json:"id"`
	OrderID        string                `json:"orderId"`
	ConfirmationNo string                `json:"confirmationNo"`
	BusinessDate   BusinessDate          `json:"businessDate"`
	ShiftID        string                `json:"shiftId,omitempty"`
	YieldQty       Dec                   `json:"yieldQty"`
	ScrapQty       Dec                   `json:"scrapQty"`
	ReworkQty      Dec                   `json:"reworkQty"`
	LabourHours    Dec                   `json:"labourHours"`
	MachineHours   Dec                   `json:"machineHours"`
	BatchID        string                `json:"batchId,omitempty"`
	ReversalOf     string                `json:"reversalOf,omitempty"`
	Reversed       bool                  `json:"reversed"`
	ReasonCode     string                `json:"reasonCode,omitempty"`
	Consumptions   []MaterialConsumption `json:"consumptions,omitempty"`
	AuditFields
}

// MaterialConsumption is one component consumed by a confirmation.
type MaterialConsumption struct {
	ID             string    `json:"id"`
	ConfirmationID string    `json:"confirmationId"`
	MaterialID     string    `json:"materialId"`
	Quantity       Dec       `json:"quantity"`
	UOM            string    `json:"uom"`
	CreatedAt      time.Time `json:"createdAt"`
	CreatedBy      string    `json:"createdBy"`
}

// ValidateConfirmation applies the rules a confirmation has to satisfy before
// anything is posted.
func ValidateConfirmation(o ProductionOrder, c ProductionConfirmation) error {
	verr := &ValidationError{}

	switch o.Status {
	case OrderReleased, OrderInProcess, OrderPartiallyConfirmed:
		// fine
	default:
		return fmt.Errorf("%w: order %s is %s and cannot be confirmed",
			ErrStateTransition, o.OrderNo, o.Status)
	}
	if !c.BusinessDate.Valid() {
		verr.Add("businessDate", "INVALID_DATE", "the confirmation needs a valid business date")
	}
	for field, value := range map[string]Dec{
		"yieldQty": c.YieldQty, "scrapQty": c.ScrapQty, "reworkQty": c.ReworkQty,
		"labourHours": c.LabourHours, "machineHours": c.MachineHours,
	} {
		if value.IsNegative() {
			verr.Add(field, "NEGATIVE", fmt.Sprintf("%s cannot be negative", field))
		}
	}
	if c.YieldQty.IsZero() && c.ScrapQty.IsZero() && c.ReworkQty.IsZero() {
		verr.Add("yieldQty", "EMPTY",
			"a confirmation must record some yield, scrap or rework")
	}
	for i, consumption := range c.Consumptions {
		if consumption.MaterialID == "" {
			verr.AddRow(i, "materialId", "REQUIRED", "a consumption line needs a material")
		}
		if consumption.Quantity.IsNegative() {
			verr.AddRow(i, "quantity", "NEGATIVE", "a consumption quantity cannot be negative")
		}
	}
	return verr.OrNil()
}

// Batch identifies a produced lot for traceability.
type Batch struct {
	ID         string       `json:"id"`
	Code       string       `json:"code"`
	ProductID  string       `json:"productId"`
	FactoryID  string       `json:"factoryId"`
	ProducedOn BusinessDate `json:"producedOn,omitempty"`
	Quantity   Dec          `json:"quantity"`
	Status     string       `json:"status"`
	AuditFields
}

// ---------------------------------------------------------------------------
// Quality
// ---------------------------------------------------------------------------

// QualityParameter is a measurable property: Brix, Pol, purity, moisture,
// colour, or anything the site defines.
type QualityParameter struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	UOM        string `json:"uom,omitempty"`
	TestMethod string `json:"testMethod,omitempty"`
	Validity
	AuditFields
}

// QualitySpec is the specification for one parameter of one product, effective
// dated so that tightening a limit does not retrospectively fail last season's
// production.
type QualitySpec struct {
	ID          string       `json:"id"`
	ProductID   string       `json:"productId"`
	ParameterID string       `json:"parameterId"`
	LowerLimit  *Dec         `json:"lowerLimit,omitempty"`
	UpperLimit  *Dec         `json:"upperLimit,omitempty"`
	WarnLower   *Dec         `json:"warnLower,omitempty"`
	WarnUpper   *Dec         `json:"warnUpper,omitempty"`
	ValidFrom   BusinessDate `json:"validFrom"`
	ValidTo     BusinessDate `json:"validTo,omitempty"`
	AuditFields
}

// IsEffectiveOn reports whether the specification applies on a date.
func (s QualitySpec) IsEffectiveOn(on BusinessDate) bool {
	if s.ValidFrom != "" && on < s.ValidFrom {
		return false
	}
	if s.ValidTo != "" && on > s.ValidTo {
		return false
	}
	return true
}

// QualitySample is material taken for testing.
type QualitySample struct {
	ID           string          `json:"id"`
	SampleNo     string          `json:"sampleNo"`
	ProductID    string          `json:"productId"`
	BatchID      string          `json:"batchId,omitempty"`
	FactoryID    string          `json:"factoryId"`
	BusinessDate BusinessDate    `json:"businessDate"`
	ShiftID      string          `json:"shiftId,omitempty"`
	TakenAt      time.Time       `json:"takenAt"`
	LabUser      string          `json:"labUser,omitempty"`
	Status       string          `json:"status"`
	Comment      string          `json:"comment,omitempty"`
	Results      []QualityResult `json:"results,omitempty"`
	AuditFields
}

// Verdict is the sample's overall outcome: the worst of its results.
func (s QualitySample) Verdict() QualityStatus {
	verdict := QualityPass
	for _, r := range s.Results {
		switch r.Status {
		case QualityFail:
			return QualityFail
		case QualityWarning:
			verdict = QualityWarning
		}
	}
	return verdict
}

// QualityResult is one measurement, with the limits that judged it copied onto
// the record.
//
// Copying the limits rather than pointing at the specification is deliberate: a
// result has to keep saying what it was judged against, even after somebody
// edits the specification.
type QualityResult struct {
	ID          string        `json:"id"`
	SampleID    string        `json:"sampleId"`
	ParameterID string        `json:"parameterId"`
	Value       Dec           `json:"value"`
	UOM         string        `json:"uom,omitempty"`
	LowerLimit  *Dec          `json:"lowerLimit,omitempty"`
	UpperLimit  *Dec          `json:"upperLimit,omitempty"`
	Status      QualityStatus `json:"status"`
	Comment     string        `json:"comment,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	CreatedBy   string        `json:"createdBy"`
}

// EvaluateResult judges a measurement against a specification.
//
//	outside a hard limit          -> FAIL
//	outside a warning band        -> WARNING
//	otherwise                     -> PASS
//
// A specification with no limits at all cannot fail anything, so the result is
// recorded as a pass; that is a configuration gap, not a quality event, and the
// service reports the gap separately.
func EvaluateResult(value Dec, spec QualitySpec) QualityStatus {
	if spec.LowerLimit != nil && value.LessThan(*spec.LowerLimit) {
		return QualityFail
	}
	if spec.UpperLimit != nil && value.GreaterThan(*spec.UpperLimit) {
		return QualityFail
	}
	if spec.WarnLower != nil && value.LessThan(*spec.WarnLower) {
		return QualityWarning
	}
	if spec.WarnUpper != nil && value.GreaterThan(*spec.WarnUpper) {
		return QualityWarning
	}
	return QualityPass
}

// QualityHold blocks a quantity from shipment and consumption until an
// authorised user releases it.
type QualityHold struct {
	ID          string       `json:"id"`
	WarehouseID string       `json:"warehouseId"`
	ProductID   string       `json:"productId"`
	BatchID     string       `json:"batchId,omitempty"`
	SampleID    string       `json:"sampleId,omitempty"`
	Quantity    Dec          `json:"quantity"`
	PlacedOn    BusinessDate `json:"placedOn"`
	ReleasedOn  BusinessDate `json:"releasedOn,omitempty"`
	ReleasedBy  string       `json:"releasedBy,omitempty"`
	Reason      string       `json:"reason"`
	AuditFields
}

// IsOpen reports whether the hold is still blocking stock.
func (h QualityHold) IsOpen() bool { return h.ReleasedOn == "" }

// ---------------------------------------------------------------------------
// Maintenance
// ---------------------------------------------------------------------------

// MaintenanceWindow is a planned outage. Once approved it removes capacity from
// the plan: the generator treats its dates as non-working.
type MaintenanceWindow struct {
	ID          string       `json:"id"`
	FactoryID   string       `json:"factoryId"`
	LineID      string       `json:"lineId,omitempty"`
	EquipmentID string       `json:"equipmentId,omitempty"`
	StartDate   BusinessDate `json:"startDate"`
	EndDate     BusinessDate `json:"endDate"`
	Description string       `json:"description,omitempty"`
	Status      string       `json:"status"`
	ApprovedBy  string       `json:"approvedBy,omitempty"`
	AuditFields
}

// Maintenance window statuses.
const (
	MaintenancePlanned   = "PLANNED"
	MaintenanceApproved  = "APPROVED"
	MaintenanceDone      = "DONE"
	MaintenanceCancelled = "CANCELLED"
)

// Dates lists every calendar day the window covers.
func (m MaintenanceWindow) Dates() []BusinessDate {
	if !m.StartDate.Valid() || !m.EndDate.Valid() || m.EndDate < m.StartDate {
		return nil
	}
	var out []BusinessDate
	for d := m.StartDate; d <= m.EndDate; d = d.AddDays(1) {
		out = append(out, d)
		if len(out) > 3650 { // a decade of shutdown is a data error, not a plan
			break
		}
	}
	return out
}

// NonWorkingDays collapses approved maintenance windows into the set of dates
// the plan generator must skip.
//
// Only approved windows count. A window somebody is still thinking about must
// not quietly move the end of the season.
func NonWorkingDays(windows []MaintenanceWindow) map[BusinessDate]bool {
	out := map[BusinessDate]bool{}
	for _, w := range windows {
		if w.Status != MaintenanceApproved {
			continue
		}
		// A line-specific outage does not stop the factory; only a
		// factory-wide window removes a crushing day.
		if w.LineID != "" {
			continue
		}
		for _, d := range w.Dates() {
			out[d] = true
		}
	}
	return out
}
