package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// This file is the production order life cycle: creating orders, releasing
// them to the floor, confirming what a shift actually made, reversing a
// confirmation that was wrong, and closing an order whose quantity has been
// reconciled.

// DefaultVarianceTolerancePct is how far a confirmed quantity may sit from the
// planned one before closing the order needs an explained, authorised variance.
// It is a constant rather than a setting because it is a default: a site that
// wants a different figure sets the CLOSE_VARIANCE_TOLERANCE_PCT assumption on
// the plan version the order came from.
var DefaultVarianceTolerancePct = domain.D("5")

// AsmCloseVarianceTolerancePct is the assumption that overrides the default.
const AsmCloseVarianceTolerancePct = "CLOSE_VARIANCE_TOLERANCE_PCT"

// ---------------------------------------------------------------------------
// Reads
// ---------------------------------------------------------------------------

// ListOrders returns the production orders the caller may see.
func (e *Execution) ListOrders(ctx context.Context, f store.ExecutionFilter) (store.Page[domain.ProductionOrder], error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return store.Page[domain.ProductionOrder]{}, err
	}
	if err := e.requireScope(caller, f.FactoryID); err != nil {
		return store.Page[domain.ProductionOrder]{}, err
	}
	page, err := e.store.Execution().ListOrders(ctx, f)
	if err != nil {
		return page, err
	}
	filtered := page.Items[:0]
	for _, o := range page.Items {
		if caller.CanSeeFactory(o.FactoryID) {
			filtered = append(filtered, o)
		}
	}
	page.Items = filtered
	page.Count = len(filtered)
	return page, nil
}

// OrderDetail is one order with everything a screen needs to act on it.
type OrderDetail struct {
	Order          domain.ProductionOrder          `json:"order"`
	Confirmations  []domain.ProductionConfirmation `json:"confirmations"`
	OpenQty        domain.Dec                      `json:"openQty"`
	VariancePct    domain.Dec                      `json:"variancePct"`
	AllowedActions []domain.OrderAction            `json:"allowedActions"`
}

// GetOrder reads one order with its confirmations.
func (e *Execution) GetOrder(ctx context.Context, id string) (OrderDetail, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return OrderDetail{}, err
	}
	order, err := e.store.Execution().GetOrder(ctx, id)
	if err != nil {
		return OrderDetail{}, err
	}
	if err := caller.RequireFactory(order.FactoryID); err != nil {
		return OrderDetail{}, err
	}
	confirmations, err := e.store.Execution().ListConfirmations(ctx, id)
	if err != nil {
		return OrderDetail{}, err
	}
	return OrderDetail{
		Order: order, Confirmations: confirmations,
		OpenQty: order.OpenQty(), VariancePct: order.VariancePct(),
		AllowedActions: domain.AllowedOrderActions(order),
	}, nil
}

// ---------------------------------------------------------------------------
// Creating orders
// ---------------------------------------------------------------------------

// OrderRequest creates one production order by hand.
type OrderRequest struct {
	FactoryID    string              `json:"factoryId"`
	LineID       string              `json:"lineId,omitempty"`
	VersionID    string              `json:"versionId,omitempty"`
	BusinessDate domain.BusinessDate `json:"businessDate"`
	ShiftID      string              `json:"shiftId,omitempty"`
	ProductID    string              `json:"productId"`
	PackagingID  string              `json:"packagingId,omitempty"`
	PlannedQty   domain.Dec          `json:"plannedQty"`
	Priority     int                 `json:"priority,omitempty"`
	Team         string              `json:"team,omitempty"`
}

// CreateOrder creates a single order.
func (e *Execution) CreateOrder(ctx context.Context, req OrderRequest) (domain.ProductionOrder, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermActualProduce); err != nil {
		return domain.ProductionOrder{}, err
	}
	if err := caller.RequireFactory(req.FactoryID); err != nil {
		return domain.ProductionOrder{}, err
	}

	verr := &domain.ValidationError{}
	if !req.BusinessDate.Valid() {
		verr.Add("businessDate", "INVALID_DATE", "the order needs a valid business date")
	}
	if req.ProductID == "" {
		verr.Add("productId", "REQUIRED", "the order needs a product")
	}
	if req.PlannedQty.LessThanOrEqual(domain.Zero) {
		verr.Add("plannedQty", "NOT_POSITIVE", "the planned quantity must be more than zero")
	}
	if req.Priority != 0 && (req.Priority < 1 || req.Priority > 9) {
		verr.Add("priority", "OUT_OF_RANGE", "the priority runs from 1 (highest) to 9 (lowest)")
	}
	if err := verr.OrNil(); err != nil {
		return domain.ProductionOrder{}, err
	}

	factory, err := e.store.MasterData().Factories().Get(ctx, req.FactoryID)
	if err != nil {
		return domain.ProductionOrder{}, err
	}

	var saved domain.ProductionOrder
	err = e.store.InTx(ctx, func(tx store.Store) error {
		order, err := e.buildOrder(ctx, tx, factory, req)
		if err != nil {
			return err
		}
		saved, err = tx.Execution().SaveOrder(ctx, order, caller.Username)
		if err != nil {
			return err
		}
		return e.audit(ctx, tx, auditEntry{
			action: "CREATE", entity: "production_order", entityID: saved.ID, after: saved,
		})
	})
	if err != nil {
		return domain.ProductionOrder{}, err
	}
	return saved, nil
}

func (e *Execution) buildOrder(ctx context.Context, tx store.Store,
	factory domain.Factory, req OrderRequest,
) (domain.ProductionOrder, error) {
	year, err := yearOf(req.BusinessDate)
	if err != nil {
		return domain.ProductionOrder{}, err
	}
	orderNo, err := tx.Execution().NextNumber(ctx, store.SeriesOrder, factory.Code, year)
	if err != nil {
		return domain.ProductionOrder{}, err
	}
	return domain.ProductionOrder{
		OrderNo: orderNo, CompanyID: factory.CompanyID, FactoryID: factory.ID,
		LineID: req.LineID, VersionID: req.VersionID, BusinessDate: req.BusinessDate,
		ShiftID: req.ShiftID, ProductID: req.ProductID, PackagingID: req.PackagingID,
		PlannedQty: domain.RoundQty(req.PlannedQty), ConfirmedQty: domain.Zero,
		Priority: req.Priority, Status: domain.OrderPlanned, Team: req.Team,
	}, nil
}

// OrdersFromPlanRequest asks for the released plan to be turned into orders for
// a range of days.
type OrdersFromPlanRequest struct {
	From domain.BusinessDate `json:"from"`
	To   domain.BusinessDate `json:"to"`
	// Products restricts the run to particular products; empty means all of
	// them.
	Products []string `json:"products,omitempty"`
}

// OrdersFromPlanResult reports what the run created and what it left alone.
type OrdersFromPlanResult struct {
	Created []domain.ProductionOrder `json:"created"`
	Skipped []SkippedRow             `json:"skipped,omitempty"`
}

// SkippedRow explains one planned row that did not become an order.
type SkippedRow struct {
	BusinessDate domain.BusinessDate `json:"businessDate"`
	ProductID    string              `json:"productId"`
	Reason       string              `json:"reason"`
}

// CreateOrdersFromPlan turns the released daily production plan into orders.
//
// Only a released version may be used: an order is an instruction to the floor,
// and instructions do not come from a draft somebody is still editing. A day
// and product that already has an order is skipped rather than duplicated, so
// the run can be repeated safely as the season advances.
func (e *Execution) CreateOrdersFromPlan(ctx context.Context, versionID string,
	req OrdersFromPlanRequest,
) (OrdersFromPlanResult, error) {

	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermActualProduce); err != nil {
		return OrdersFromPlanResult{}, err
	}

	version, err := e.store.Planning().GetVersion(ctx, versionID)
	if err != nil {
		return OrdersFromPlanResult{}, err
	}
	season, err := e.store.Planning().GetSeason(ctx, version.SeasonID)
	if err != nil {
		return OrdersFromPlanResult{}, err
	}
	if err := caller.RequireFactory(season.FactoryID); err != nil {
		return OrdersFromPlanResult{}, err
	}
	if version.Status != domain.StatusReleased {
		return OrdersFromPlanResult{}, fmt.Errorf(
			"%w: version %s is %s; only a released plan may be turned into production orders",
			domain.ErrStateTransition, version.Code, version.Status)
	}
	// Every season has an actuals container, and it is released from the day it
	// is created so operators can post to it. It is not a plan: it records what
	// happened, so there is nothing in it to instruct the floor with.
	if version.PlanType == domain.PlanTypeActual {
		return OrdersFromPlanResult{}, fmt.Errorf(
			"%w: %s is the actuals container, which records what happened rather than what to make; "+
				"raise the orders from the released plan instead",
			domain.ErrValidation, version.Code)
	}
	if !req.From.Valid() || !req.To.Valid() {
		return OrdersFromPlanResult{}, fmt.Errorf(
			"%w: the run needs a valid from and to date", domain.ErrValidation)
	}
	if req.To < req.From {
		return OrdersFromPlanResult{}, fmt.Errorf(
			"%w: the end of the range is before its start", domain.ErrValidation)
	}

	rows, err := e.store.Planning().ListProducts(ctx, store.PlanFilter{
		VersionIDs: []string{versionID}, From: req.From, To: req.To,
		ProductIDs: req.Products, Series: domain.SeriesPlan,
	})
	if err != nil {
		return OrdersFromPlanResult{}, err
	}

	existing, err := e.store.Execution().ListOrders(ctx, store.ExecutionFilter{
		FactoryID: season.FactoryID, From: req.From, To: req.To, Top: 5000,
	})
	if err != nil {
		return OrdersFromPlanResult{}, err
	}
	covered := map[string]bool{}
	for _, o := range existing.Items {
		if o.Status == domain.OrderCancelled {
			continue
		}
		covered[orderKey(o.BusinessDate, o.ProductID, o.LineID, o.ShiftID)] = true
	}

	factory, err := e.store.MasterData().Factories().Get(ctx, season.FactoryID)
	if err != nil {
		return OrdersFromPlanResult{}, err
	}

	result := OrdersFromPlanResult{}
	err = e.store.InTx(ctx, func(tx store.Store) error {
		for _, row := range rows {
			if row.Quantity.LessThanOrEqual(domain.Zero) {
				result.Skipped = append(result.Skipped, SkippedRow{
					BusinessDate: row.BusinessDate, ProductID: row.ProductID,
					Reason: "the plan has nothing to produce on this day",
				})
				continue
			}
			key := orderKey(row.BusinessDate, row.ProductID, row.LineID, row.ShiftID)
			if covered[key] {
				result.Skipped = append(result.Skipped, SkippedRow{
					BusinessDate: row.BusinessDate, ProductID: row.ProductID,
					Reason: "an order already covers this day and product",
				})
				continue
			}

			order, err := e.buildOrder(ctx, tx, factory, OrderRequest{
				FactoryID: season.FactoryID, LineID: row.LineID, VersionID: versionID,
				BusinessDate: row.BusinessDate, ShiftID: row.ShiftID, ProductID: row.ProductID,
				PackagingID: row.PackagingID, PlannedQty: row.Quantity,
			})
			if err != nil {
				return err
			}
			saved, err := tx.Execution().SaveOrder(ctx, order, caller.Username)
			if err != nil {
				return err
			}
			covered[key] = true
			result.Created = append(result.Created, saved)
		}
		if len(result.Created) == 0 {
			return nil
		}
		return e.audit(ctx, tx, auditEntry{
			action: "CREATE_FROM_PLAN", entity: "plan_version", entityID: versionID,
			after: result.Created,
			reason: fmt.Sprintf("created %d production orders for %s to %s",
				len(result.Created), req.From, req.To),
		})
	})
	if err != nil {
		return OrdersFromPlanResult{}, err
	}
	return result, nil
}

func orderKey(date domain.BusinessDate, product, line, shift string) string {
	return string(date) + "|" + product + "|" + line + "|" + shift
}

// ---------------------------------------------------------------------------
// Order actions
// ---------------------------------------------------------------------------

// OrderActionRequest applies a command to an order.
type OrderActionRequest struct {
	Action domain.OrderAction `json:"action"`
	Reason string             `json:"reason,omitempty"`
	// VarianceReason is a reason code, required when closing an order whose
	// confirmed quantity is outside the tolerance.
	VarianceReason string `json:"varianceReason,omitempty"`
	RowVersion     int64  `json:"rowVersion"`
}

// ActOnOrder moves an order through its life cycle.
func (e *Execution) ActOnOrder(ctx context.Context, id string, req OrderActionRequest) (domain.ProductionOrder, error) {
	caller := auth.FromContext(ctx)

	order, err := e.store.Execution().GetOrder(ctx, id)
	if err != nil {
		return domain.ProductionOrder{}, err
	}
	if err := caller.RequireFactory(order.FactoryID); err != nil {
		return domain.ProductionOrder{}, err
	}
	if req.RowVersion != 0 && req.RowVersion != order.RowVersion {
		return domain.ProductionOrder{}, fmt.Errorf(
			"%w: order %s is at version %d, you have %d",
			domain.ErrConflict, order.OrderNo, order.RowVersion, req.RowVersion)
	}

	next, permission, err := domain.ApplyOrderAction(order, req.Action)
	if err != nil {
		return domain.ProductionOrder{}, err
	}
	if err := caller.Require(permission); err != nil {
		return domain.ProductionOrder{}, err
	}

	if req.Action == domain.OrderActionClose {
		tolerance, err := e.varianceTolerance(ctx, order)
		if err != nil {
			return domain.ProductionOrder{}, err
		}
		// Closing an order that did not make what it was told to make is an
		// explained act: the reason is recorded on the order, and an approver
		// has to be the one recording it.
		if err := domain.CheckOrderClose(order, tolerance, req.VarianceReason,
			caller.Can(domain.PermPlanApprove)); err != nil {
			return domain.ProductionOrder{}, err
		}
		if req.VarianceReason != "" {
			if _, err := e.store.MasterData().ReasonCodes().GetByCode(ctx, req.VarianceReason); err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					return domain.ProductionOrder{}, fmt.Errorf(
						"%w: %q is not a reason code", domain.ErrValidation, req.VarianceReason)
				}
				return domain.ProductionOrder{}, err
			}
			order.VarianceReason = req.VarianceReason
		}
	}

	before := order
	order.Status = next

	var saved domain.ProductionOrder
	err = e.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Execution().SaveOrder(ctx, order, caller.Username)
		if err != nil {
			return err
		}
		return e.audit(ctx, tx, auditEntry{
			action: string(req.Action), entity: "production_order", entityID: saved.ID,
			before: before, after: saved, reason: req.Reason,
		})
	})
	if err != nil {
		return domain.ProductionOrder{}, err
	}
	return saved, nil
}

// varianceTolerance reads the closing tolerance from the plan version the order
// came from, falling back to the default.
func (e *Execution) varianceTolerance(ctx context.Context, order domain.ProductionOrder) (domain.Dec, error) {
	if order.VersionID == "" {
		return DefaultVarianceTolerancePct, nil
	}
	assumptions, err := e.store.Planning().ListAssumptions(ctx, order.VersionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return DefaultVarianceTolerancePct, nil
		}
		return domain.Zero, err
	}
	for _, a := range assumptions {
		if a.Code == AsmCloseVarianceTolerancePct && !a.Value.IsNegative() {
			return a.Value, nil
		}
	}
	return DefaultVarianceTolerancePct, nil
}

// ---------------------------------------------------------------------------
// Confirmations
// ---------------------------------------------------------------------------

// ConfirmRequest records what a shift produced.
type ConfirmRequest struct {
	BusinessDate domain.BusinessDate `json:"businessDate"`
	ShiftID      string              `json:"shiftId,omitempty"`
	YieldQty     domain.Dec          `json:"yieldQty"`
	ScrapQty     domain.Dec          `json:"scrapQty,omitempty"`
	ReworkQty    domain.Dec          `json:"reworkQty,omitempty"`
	LabourHours  domain.Dec          `json:"labourHours,omitempty"`
	MachineHours domain.Dec          `json:"machineHours,omitempty"`
	// BatchID names an existing batch; BatchCode names one by the code written
	// on the pallet card, creating it if the shift is the first to use it.
	// Production is what creates a batch, so a confirmation is exactly the
	// place it comes into existence.
	BatchID    string `json:"batchId,omitempty"`
	BatchCode  string `json:"batchCode,omitempty"`
	ReasonCode string `json:"reasonCode,omitempty"`
	// WarehouseID receives the yield. A confirmation without one records the
	// production but moves no stock, which is what a site that keeps its
	// inventory elsewhere wants.
	WarehouseID  string                     `json:"warehouseId,omitempty"`
	Consumptions []ConsumptionInput         `json:"consumptions,omitempty"`
	Overrides    ConfirmationOverrideOption `json:"overrides,omitempty"`
}

// ConsumptionInput is one component a confirmation consumed.
type ConsumptionInput struct {
	MaterialID string     `json:"materialId"`
	Quantity   domain.Dec `json:"quantity"`
	UOM        string     `json:"uom,omitempty"`
}

// ConfirmationOverrideOption carries the same authorisations a hand posting
// can ask for, for the receipt the confirmation generates.
type ConfirmationOverrideOption struct {
	CapacityOverride bool `json:"capacityOverride,omitempty"`
}

// ConfirmResult is what a confirmation did.
type ConfirmResult struct {
	Confirmation domain.ProductionConfirmation `json:"confirmation"`
	Order        domain.ProductionOrder        `json:"order"`
	Document     *domain.InventoryDocument     `json:"document,omitempty"`
}

// Confirm records production against an order and receipts the yield into
// stock.
//
// The confirmation, the order's new quantity and status, the goods receipt and
// the audit record are written in one transaction. A confirmation that fails
// the stock rules - typically because the receiving store is full - posts
// nothing at all, rather than leaving a confirmed order whose sugar is nowhere.
func (e *Execution) Confirm(ctx context.Context, orderID string, req ConfirmRequest) (ConfirmResult, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermActualProduce); err != nil {
		return ConfirmResult{}, err
	}

	order, err := e.store.Execution().GetOrder(ctx, orderID)
	if err != nil {
		return ConfirmResult{}, err
	}
	if err := caller.RequireFactory(order.FactoryID); err != nil {
		return ConfirmResult{}, err
	}

	if req.BusinessDate == "" {
		req.BusinessDate = order.BusinessDate
	}
	confirmation := domain.ProductionConfirmation{
		OrderID: order.ID, BusinessDate: req.BusinessDate, ShiftID: req.ShiftID,
		YieldQty: domain.RoundQty(req.YieldQty), ScrapQty: domain.RoundQty(req.ScrapQty),
		ReworkQty:   domain.RoundQty(req.ReworkQty),
		LabourHours: domain.RoundRate(req.LabourHours), MachineHours: domain.RoundRate(req.MachineHours),
		BatchID: req.BatchID, ReasonCode: req.ReasonCode,
	}
	if confirmation.ShiftID == "" {
		confirmation.ShiftID = order.ShiftID
	}
	if confirmation.BatchID == "" && req.BatchCode == "" {
		confirmation.BatchID = order.BatchID
	}
	for _, c := range req.Consumptions {
		confirmation.Consumptions = append(confirmation.Consumptions, domain.MaterialConsumption{
			MaterialID: c.MaterialID, Quantity: domain.RoundQty(c.Quantity), UOM: c.UOM,
		})
	}
	// A confirmation that names no components gets them from the bill of
	// materials of the packaging it produced.
	//
	// This is not a convenience. An operator confirming a shift at the end of it
	// is not going to key how many liners and how much thread went into 300
	// jumbo bags, so asked for them by hand they would simply not be recorded -
	// and a material consumption nobody records is a stock figure that drifts
	// until somebody counts the shed. Derived from the bill of materials it is
	// at least a defensible number, and an operator who knows better can still
	// send the real ones.
	if len(confirmation.Consumptions) == 0 {
		derived, err := e.consumptionFromBOM(ctx, order, confirmation.YieldQty)
		if err != nil {
			return ConfirmResult{}, err
		}
		confirmation.Consumptions = derived
	}
	if err := domain.ValidateConfirmation(order, confirmation); err != nil {
		return ConfirmResult{}, err
	}

	opts := domain.PostingOptions{}
	if req.Overrides.CapacityOverride {
		if err := caller.Require(domain.PermCapacityOverride); err != nil {
			return ConfirmResult{}, err
		}
		opts.CapacityOverride = true
	}

	factory, err := e.store.MasterData().Factories().Get(ctx, order.FactoryID)
	if err != nil {
		return ConfirmResult{}, err
	}
	product, err := e.store.MasterData().Products().Get(ctx, order.ProductID)
	if err != nil {
		return ConfirmResult{}, err
	}
	year, err := yearOf(confirmation.BusinessDate)
	if err != nil {
		return ConfirmResult{}, err
	}

	before := order
	result := ConfirmResult{}
	err = e.store.InTx(ctx, func(tx store.Store) error {
		// The batch is resolved inside the transaction so that a batch created
		// by a confirmation that then fails does not survive it.
		confirmation.BatchID, err = e.resolveBatch(ctx, tx, "batchId",
			confirmation.BatchID, req.BatchCode, order.ProductID, order.FactoryID,
			confirmation.BusinessDate, true)
		if err != nil {
			return err
		}

		confirmation.ConfirmationNo, err = tx.Execution().NextNumber(
			ctx, store.SeriesConfirmation, factory.Code, year)
		if err != nil {
			return err
		}
		result.Confirmation, err = tx.Execution().SaveConfirmation(ctx, confirmation, caller.Username)
		if err != nil {
			return err
		}

		// The good output goes into stock. Scrap and rework do not: scrap has
		// left the process and rework has not finished it, and inventing a
		// balance for either would put sugar in a shed that does not hold any.
		if req.WarehouseID != "" && confirmation.YieldQty.GreaterThan(domain.Zero) {
			document, err := e.postChecked(ctx, tx, domain.InventoryDocument{
				DocType: domain.DocReceipt, BusinessDate: confirmation.BusinessDate,
				FactoryID: order.FactoryID, Reference: confirmation.ConfirmationNo,
				Note: "goods receipt from order " + order.OrderNo,
				Items: []domain.InventoryDocumentItem{{
					LineNo: 1, WarehouseID: req.WarehouseID, ProductID: order.ProductID,
					BatchID: confirmation.BatchID, Quantity: confirmation.YieldQty,
					UOM: product.BaseUOM,
				}},
			}, opts)
			if err != nil {
				return err
			}
			result.Document = &document
		}

		order.ConfirmedQty = domain.RoundQty(order.ConfirmedQty.Add(confirmation.YieldQty))
		order.Status = domain.ConfirmedStatus(order.PlannedQty, order.ConfirmedQty)
		if confirmation.BatchID != "" && order.BatchID == "" {
			order.BatchID = confirmation.BatchID
		}
		result.Order, err = tx.Execution().SaveOrder(ctx, order, caller.Username)
		if err != nil {
			return err
		}

		if err := e.audit(ctx, tx, auditEntry{
			action: "CONFIRM", entity: "production_order", entityID: order.ID,
			before: before, after: result.Order,
			reason: fmt.Sprintf("confirmation %s: %s t", confirmation.ConfirmationNo, confirmation.YieldQty),
		}); err != nil {
			return err
		}

		return emit(ctx, tx, domain.TopicProductionConfirm, ConfirmationEvent{
			ConfirmationID: result.Confirmation.ID, ConfirmationNo: result.Confirmation.ConfirmationNo,
			OrderID: order.ID, OrderNo: order.OrderNo, FactoryID: order.FactoryID,
			BusinessDate: confirmation.BusinessDate, ProductID: order.ProductID,
			BatchID: result.Confirmation.BatchID, GoodQuantity: confirmation.YieldQty,
			ScrapQuantity: confirmation.ScrapQty, ReworkQuantity: confirmation.ReworkQty,
			OrderStatus: result.Order.Status,
			// A consumer that closes its own order needs to know this was the
			// last confirmation, and the order's status is what says so.
			Final:       result.Order.Status == domain.OrderCompleted,
			ConfirmedBy: caller.Username,
		})
	})
	if err != nil {
		return ConfirmResult{}, err
	}
	return result, nil
}

// resolveBatch turns whatever the caller named into the id of a batch row.
//
// A batch is a row rather than a string typed twice: it is what a certificate
// of analysis is about, what a hold blocks and what a customer quotes back when
// something is wrong with a consignment. A free-text batch would make all three
// unanswerable.
//
// create says whether the caller is entitled to bring a batch into existence.
// Production is; a laboratory sample is not, because a sample of a batch nobody
// produced is a sample of nothing.
func (e *Execution) resolveBatch(ctx context.Context, tx store.Store, field, id, code string,
	productID, factoryID string, on domain.BusinessDate, create bool,
) (string, error) {

	if id != "" {
		batch, err := tx.Execution().GetBatch(ctx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return "", fmt.Errorf("%w: %s names a batch that does not exist",
					domain.ErrValidation, field)
			}
			return "", err
		}
		return batch.ID, nil
	}
	if code == "" {
		return "", nil
	}

	batch, err := tx.Execution().BatchByCode(ctx, code)
	if err == nil {
		return batch.ID, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return "", err
	}
	if !create {
		return "", fmt.Errorf("%w: no batch is coded %s; it is created by the production "+
			"that made it, not here", domain.ErrValidation, code)
	}

	saved, err := tx.Execution().SaveBatch(ctx, domain.Batch{
		Code: code, ProductID: productID, FactoryID: factoryID, ProducedOn: on, Status: "OPEN",
	}, auth.FromContext(ctx).Username)
	if err != nil {
		return "", err
	}
	return saved.ID, nil
}

// consumptionFromBOM works out what a yield consumed, from the packaging type
// the order produces and its bill of materials.
//
// An order with no packaging - bulk raw sugar to the refinery - consumes no
// packaging, and this returns nothing rather than guessing at a bag.
func (e *Execution) consumptionFromBOM(ctx context.Context, order domain.ProductionOrder,
	yield domain.Dec,
) ([]domain.MaterialConsumption, error) {

	if order.PackagingID == "" || yield.LessThanOrEqual(domain.Zero) {
		return nil, nil
	}
	md := e.store.MasterData()

	packaging, err := md.PackagingTypes().Get(ctx, order.PackagingID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	materials := map[string]domain.Material{}
	load := func(id string) (domain.Material, bool, error) {
		if id == "" {
			return domain.Material{}, false, nil
		}
		if mat, ok := materials[id]; ok {
			return mat, true, nil
		}
		mat, err := md.Materials().Get(ctx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.Material{}, false, nil
			}
			return domain.Material{}, false, err
		}
		materials[id] = mat
		return mat, true, nil
	}

	primary, hasPrimary, err := load(packaging.MaterialID)
	if err != nil {
		return nil, err
	}
	// The scrap allowance is the bag's, because it is the bag count that the
	// components are derived from.
	packages := domain.RequiredPackages(yield, packaging.NetWeightKg, primary.ScrapPct)
	if packages <= 0 {
		return nil, nil
	}

	var out []domain.MaterialConsumption
	if hasPrimary {
		out = append(out, domain.MaterialConsumption{
			MaterialID: primary.ID, Quantity: domain.DI(packages), UOM: primary.UOM,
		})
	}

	lines, err := md.ListPackagingBOM(ctx, packaging.ID)
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		if !line.Active {
			continue
		}
		component, ok, err := load(line.MaterialID)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		quantity := domain.ComponentQuantity(packages, line.QtyPerPackage, component.ScrapPct)
		if quantity.LessThanOrEqual(domain.Zero) {
			continue
		}
		out = append(out, domain.MaterialConsumption{
			MaterialID: component.ID, Quantity: quantity, UOM: component.UOM,
		})
	}
	return out, nil
}

// ReverseConfirmation undoes a confirmation: the goods receipt is reversed, the
// order's confirmed quantity comes back down, and both records stay in place.
func (e *Execution) ReverseConfirmation(ctx context.Context, confirmationID string,
	req ReversalRequest,
) (ConfirmResult, error) {

	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermActualProduce); err != nil {
		return ConfirmResult{}, err
	}
	if req.Reason == "" {
		return ConfirmResult{}, fmt.Errorf(
			"%w: reversing a confirmation needs a reason for the audit trail", domain.ErrValidation)
	}

	confirmation, err := e.store.Execution().GetConfirmation(ctx, confirmationID)
	if err != nil {
		return ConfirmResult{}, err
	}
	if confirmation.Reversed {
		return ConfirmResult{}, fmt.Errorf("%w: confirmation %s has already been reversed",
			domain.ErrValidation, confirmation.ConfirmationNo)
	}
	order, err := e.store.Execution().GetOrder(ctx, confirmation.OrderID)
	if err != nil {
		return ConfirmResult{}, err
	}
	if err := caller.RequireFactory(order.FactoryID); err != nil {
		return ConfirmResult{}, err
	}

	// The receipt this confirmation created is found by the reference it was
	// posted with, which is the confirmation number.
	documents, err := e.store.Execution().ListDocuments(ctx, store.ExecutionFilter{
		FactoryID: order.FactoryID, Top: 5000,
	})
	if err != nil {
		return ConfirmResult{}, err
	}
	var receipt *domain.InventoryDocument
	for i, d := range documents.Items {
		if d.Reference == confirmation.ConfirmationNo && d.DocType == domain.DocReceipt && !d.Reversed {
			receipt = &documents.Items[i]
			break
		}
	}

	before := order
	result := ConfirmResult{}
	err = e.store.InTx(ctx, func(tx store.Store) error {
		if receipt != nil {
			reversal, err := domain.ReverseDocument(*receipt, req.BusinessDate, req.Reason, caller.Username)
			if err != nil {
				return err
			}
			// The sugar this receipt brought in may already have shipped, so
			// the counter-posting is allowed to take the balance negative; the
			// alternative is leaving a receipt standing that everybody agrees
			// never happened.
			document, err := e.postChecked(ctx, tx, reversal,
				domain.PostingOptions{AllowNegativeStock: true, CapacityOverride: true})
			if err != nil {
				return err
			}
			if err := tx.Execution().MarkReversed(ctx, receipt.ID, caller.Username); err != nil {
				return err
			}
			result.Document = &document
		}

		confirmation.Reversed = true
		result.Confirmation, err = tx.Execution().SaveConfirmation(ctx, confirmation, caller.Username)
		if err != nil {
			return err
		}

		order.ConfirmedQty = domain.RoundQty(
			domain.ClampNonNegative(order.ConfirmedQty.Sub(confirmation.YieldQty)))
		order.Status = domain.ConfirmedStatus(order.PlannedQty, order.ConfirmedQty)
		result.Order, err = tx.Execution().SaveOrder(ctx, order, caller.Username)
		if err != nil {
			return err
		}

		return e.audit(ctx, tx, auditEntry{
			action: "REVERSE_CONFIRMATION", entity: "production_confirmation",
			entityID: confirmation.ID, before: before, after: result.Order, reason: req.Reason,
		})
	})
	if err != nil {
		return ConfirmResult{}, err
	}
	return result, nil
}
