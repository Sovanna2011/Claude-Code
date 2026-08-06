package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Execution is the daily factory service: inventory postings, production
// orders and confirmations, quality and maintenance.
//
// It carries the same responsibilities as the planning service - authorisation,
// validation, orchestration, transactions and the audit trail - and, as there,
// every calculation and every rule belongs to the domain package. What is
// specific to this service is that its writes move stock, so almost everything
// happens inside a transaction that either posts the document, the balance and
// the audit record together or posts none of them.
type Execution struct {
	store store.Store
	now   func() time.Time
}

// NewExecution builds the service. The clock is injected so tests are
// deterministic.
func NewExecution(s store.Store, now func() time.Time) *Execution {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Execution{store: s, now: now}
}

func (e *Execution) audit(ctx context.Context, tx store.Store, entry auditEntry) error {
	return writeAudit(ctx, tx, e.now, entry)
}

// ---------------------------------------------------------------------------
// Posting requests
// ---------------------------------------------------------------------------

// PostingRequest is a request to move stock.
//
// The document number is issued by the service, not by the client: a client
// that picks its own numbers is a client that can collide with another one, and
// the number is what the whole audit trail hangs off.
type PostingRequest struct {
	DocType      domain.DocType      `json:"docType"`
	BusinessDate domain.BusinessDate `json:"businessDate"`
	FactoryID    string              `json:"factoryId"`
	Reference    string              `json:"reference,omitempty"`
	ReasonCode   string              `json:"reasonCode,omitempty"`
	Note         string              `json:"note,omitempty"`
	Lines        []PostingLineInput  `json:"lines"`
	// AllowNegativeStock and CapacityOverride ask for a rule to be broken.
	// Both are refused unless the caller holds the matching permission, and
	// both are recorded on the audit event when they are used.
	AllowNegativeStock bool `json:"allowNegativeStock,omitempty"`
	CapacityOverride   bool `json:"capacityOverride,omitempty"`
}

// PostingLineInput is one line of a posting request.
//
// The quantity is unsigned here and given its sign by the document type, which
// is how a warehouse keeper thinks: an issue of 120 t is "issue 120", not
// "add minus 120". A transfer names the receiving store and is expanded into
// the pair of signed lines that actually move the balances.
type PostingLineInput struct {
	WarehouseID string `json:"warehouseId"`
	ToWarehouse string `json:"toWarehouse,omitempty"`
	ProductID   string `json:"productId"`
	// BatchID names an existing batch; BatchCode names one by the code on the
	// pallet card. A movement cannot create a batch: stock is moved, and moving
	// it is not what brings it into existence.
	BatchID   string     `json:"batchId,omitempty"`
	BatchCode string     `json:"batchCode,omitempty"`
	Quantity  domain.Dec `json:"quantity"`
	UOM       string     `json:"uom,omitempty"`
}

// signedFor works out what a line of this document type does to a balance.
//
// Adjustments and counts are the exception: they carry their own sign, because
// a correction can go either way and forcing the keeper to choose a document
// type by the sign of the difference would be busy-work.
func signedFor(docType domain.DocType, quantity domain.Dec) (domain.Dec, error) {
	switch docType {
	case domain.DocReceipt, domain.DocHold, domain.DocRelease:
		if quantity.IsNegative() {
			return domain.Zero, fmt.Errorf("%w: a %s quantity is entered as a positive number",
				domain.ErrValidation, docType)
		}
		return quantity, nil
	case domain.DocIssue, domain.DocShipment:
		if quantity.IsNegative() {
			return domain.Zero, fmt.Errorf("%w: a %s quantity is entered as a positive number",
				domain.ErrValidation, docType)
		}
		return quantity.Neg(), nil
	case domain.DocAdjustment, domain.DocCount, domain.DocTransfer:
		return quantity, nil
	case domain.DocReversal:
		return domain.Zero, fmt.Errorf(
			"%w: a reversal is posted by reversing the original document, not by hand",
			domain.ErrValidation)
	}
	return domain.Zero, fmt.Errorf("%w: %q is not a document type", domain.ErrValidation, docType)
}

// Post checks a movement against the current position and, if it is allowed,
// writes the document, its lines and the resulting balances in one transaction.
func (e *Execution) Post(ctx context.Context, req PostingRequest) (domain.InventoryDocument, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermActualStock); err != nil {
		return domain.InventoryDocument{}, err
	}
	if err := caller.RequireFactory(req.FactoryID); err != nil {
		return domain.InventoryDocument{}, err
	}
	if !domain.ValidDocType(req.DocType) {
		return domain.InventoryDocument{}, fmt.Errorf(
			"%w: %q is not a document type", domain.ErrValidation, req.DocType)
	}
	if !req.BusinessDate.Valid() {
		return domain.InventoryDocument{}, fmt.Errorf(
			"%w: the posting needs a valid business date", domain.ErrValidation)
	}
	if len(req.Lines) == 0 {
		return domain.InventoryDocument{}, fmt.Errorf(
			"%w: a posting needs at least one line", domain.ErrValidation)
	}

	opts, err := e.postingOptions(caller, req)
	if err != nil {
		return domain.InventoryDocument{}, err
	}

	items, err := e.expandLines(ctx, req)
	if err != nil {
		return domain.InventoryDocument{}, err
	}

	document := domain.InventoryDocument{
		DocType: req.DocType, BusinessDate: req.BusinessDate, FactoryID: req.FactoryID,
		Reference: req.Reference, ReasonCode: req.ReasonCode, Note: req.Note, Items: items,
	}

	var posted domain.InventoryDocument
	err = e.store.InTx(ctx, func(tx store.Store) error {
		var err error
		posted, err = e.postChecked(ctx, tx, document, opts)
		if err != nil {
			return err
		}
		return e.audit(ctx, tx, auditEntry{
			action: "POST_" + string(req.DocType), entity: "inventory_document",
			entityID: posted.ID, after: posted, reason: overrideReason(req, req.Note),
		})
	})
	if err != nil {
		return domain.InventoryDocument{}, err
	}
	return posted, nil
}

// postingOptions turns the overrides a request asks for into the ones the
// caller is actually allowed, refusing rather than silently ignoring: a keeper
// who asked to post past capacity needs to be told they may not, not left to
// discover later that the posting was rejected for a reason they overrode.
func (e *Execution) postingOptions(caller auth.Principal, req PostingRequest) (domain.PostingOptions, error) {
	opts := domain.PostingOptions{}
	if req.CapacityOverride {
		if err := caller.Require(domain.PermCapacityOverride); err != nil {
			return opts, err
		}
		opts.CapacityOverride = true
	}
	if req.AllowNegativeStock {
		// Driving a balance below zero is an administrative act, not a
		// warehouse one, so it sits behind the same permission as any other
		// override of a stock rule.
		if err := caller.Require(domain.PermCapacityOverride); err != nil {
			return opts, fmt.Errorf(
				"%w: posting below zero stock needs the %s permission",
				domain.ErrForbidden, domain.PermCapacityOverride)
		}
		opts.AllowNegativeStock = true
	}
	return opts, nil
}

func overrideReason(req PostingRequest, note string) string {
	switch {
	case req.CapacityOverride && req.AllowNegativeStock:
		return joinReason(note, "posted with a capacity override and below zero stock")
	case req.CapacityOverride:
		return joinReason(note, "posted with a capacity override")
	case req.AllowNegativeStock:
		return joinReason(note, "posted below zero stock")
	}
	return note
}

func joinReason(note, suffix string) string {
	if note == "" {
		return suffix
	}
	return note + "; " + suffix
}

// expandLines validates the request lines and turns them into the signed
// document items the ledger stores.
func (e *Execution) expandLines(ctx context.Context, req PostingRequest) ([]domain.InventoryDocumentItem, error) {
	verr := &domain.ValidationError{}
	md := e.store.MasterData()

	var items []domain.InventoryDocumentItem
	lineNo := 0
	for i, line := range req.Lines {
		if line.WarehouseID == "" {
			verr.AddRow(i, "warehouseId", "REQUIRED", "a posting line needs a warehouse")
			continue
		}
		if line.ProductID == "" {
			verr.AddRow(i, "productId", "REQUIRED", "a posting line needs a product")
			continue
		}
		if line.Quantity.IsZero() {
			verr.AddRow(i, "quantity", "ZERO", "a posting line of zero moves nothing")
			continue
		}

		product, err := md.Products().Get(ctx, line.ProductID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				verr.AddRow(i, "productId", "NOT_FOUND", "no such product")
				continue
			}
			return nil, err
		}

		batchID, err := e.resolveBatch(ctx, e.store, "batchId", line.BatchID, line.BatchCode,
			line.ProductID, req.FactoryID, req.BusinessDate, false)
		if err != nil {
			if errors.Is(err, domain.ErrValidation) {
				verr.AddRow(i, "batchId", "NOT_FOUND", err.Error())
				continue
			}
			return nil, err
		}
		line.BatchID = batchID
		uom := line.UOM
		if uom == "" {
			uom = product.BaseUOM
		}

		signed, err := signedFor(req.DocType, line.Quantity)
		if err != nil {
			verr.AddRow(i, "quantity", "SIGN", err.Error())
			continue
		}

		if req.DocType == domain.DocTransfer {
			if line.ToWarehouse == "" {
				verr.AddRow(i, "toWarehouse", "REQUIRED", "a transfer needs a receiving warehouse")
				continue
			}
			if line.ToWarehouse == line.WarehouseID {
				verr.AddRow(i, "toWarehouse", "SAME_WAREHOUSE",
					"a transfer must move the stock somewhere else")
				continue
			}
			if line.Quantity.IsNegative() {
				verr.AddRow(i, "quantity", "SIGN",
					"a transfer quantity is entered as a positive number; the direction is the pair of warehouses")
				continue
			}
			// One request line becomes two ledger lines, so the two halves of
			// the move can never be posted apart from one another.
			lineNo++
			items = append(items, domain.InventoryDocumentItem{
				LineNo: lineNo, WarehouseID: line.WarehouseID, ProductID: line.ProductID,
				BatchID: line.BatchID, Quantity: domain.RoundQty(signed.Neg()), UOM: uom,
				ToWarehouse: line.ToWarehouse,
			})
			lineNo++
			items = append(items, domain.InventoryDocumentItem{
				LineNo: lineNo, WarehouseID: line.ToWarehouse, ProductID: line.ProductID,
				BatchID: line.BatchID, Quantity: domain.RoundQty(signed), UOM: uom,
			})
			continue
		}

		lineNo++
		items = append(items, domain.InventoryDocumentItem{
			LineNo: lineNo, WarehouseID: line.WarehouseID, ProductID: line.ProductID,
			BatchID: line.BatchID, Quantity: domain.RoundQty(signed), UOM: uom,
		})
	}
	if err := verr.OrNil(); err != nil {
		return nil, err
	}
	return items, nil
}

// postChecked reads the positions the document touches, asks the domain whether
// the movement is allowed, and posts it. It is the single door every stock
// movement in the application goes through - confirmations, shipments, quality
// holds and hand-entered corrections alike - so there is exactly one place
// where the rules can be applied and exactly one place where they can be
// forgotten.
func (e *Execution) postChecked(ctx context.Context, tx store.Store,
	document domain.InventoryDocument, opts domain.PostingOptions,
) (domain.InventoryDocument, error) {

	exec := tx.Execution()

	pairs := make([][2]string, 0, len(document.Items))
	for _, item := range document.Items {
		pairs = append(pairs, [2]string{item.WarehouseID, item.ProductID})
	}
	positions, err := exec.Positions(ctx, pairs)
	if err != nil {
		return domain.InventoryDocument{}, err
	}

	warehouses := map[string]domain.Warehouse{}
	lines := make([]domain.PostingLine, 0, len(document.Items))
	// The lines are applied to a running copy of each position, so a document
	// with two lines against the same store is judged on their combined effect
	// rather than on each one against the balance as it was before the
	// document started.
	running := map[string]domain.StockPosition{}
	for _, item := range document.Items {
		key := item.WarehouseID + "|" + item.ProductID
		if _, ok := running[key]; !ok {
			running[key] = positions[key]
		}
		warehouse, ok := warehouses[item.WarehouseID]
		if !ok {
			warehouse, err = tx.MasterData().Warehouses().Get(ctx, item.WarehouseID)
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					return domain.InventoryDocument{}, fmt.Errorf(
						"%w: line %d names a warehouse that does not exist",
						domain.ErrValidation, item.LineNo)
				}
				return domain.InventoryDocument{}, err
			}
			warehouses[item.WarehouseID] = warehouse
		}

		quantityDelta, holdDelta := item.Quantity, domain.Zero
		switch document.DocType {
		case domain.DocHold:
			quantityDelta, holdDelta = domain.Zero, item.Quantity
		case domain.DocRelease:
			quantityDelta, holdDelta = domain.Zero, item.Quantity.Neg()
		}

		lines = append(lines, domain.PostingLine{
			LineNo: item.LineNo, Warehouse: warehouse, Position: running[key],
			Delta: quantityDelta, HoldDelta: holdDelta,
		})

		position := running[key]
		position.Quantity = position.Quantity.Add(quantityDelta)
		position.HoldQuantity = position.HoldQuantity.Add(holdDelta)
		running[key] = position
	}

	if err := domain.CheckPosting(lines, opts); err != nil {
		return domain.InventoryDocument{}, err
	}

	if document.DocumentNo == "" {
		factory, err := tx.MasterData().Factories().Get(ctx, document.FactoryID)
		if err != nil {
			return domain.InventoryDocument{}, err
		}
		year, err := yearOf(document.BusinessDate)
		if err != nil {
			return domain.InventoryDocument{}, err
		}
		document.DocumentNo, err = exec.NextNumber(ctx, store.SeriesDocument, factory.Code, year)
		if err != nil {
			return domain.InventoryDocument{}, err
		}
	}

	caller := auth.FromContext(ctx)
	posted, err := exec.PostDocument(ctx, document, caller.Username)
	if err != nil {
		return domain.InventoryDocument{}, err
	}

	// Every stock movement in the application comes through here, so this is
	// also the one place the integration event has to be written. Emitting it
	// from each caller instead would mean a movement added later quietly stops
	// telling the ERP anything.
	if err := emit(ctx, tx, topicForDocument(posted.DocType), stockEvent(posted)); err != nil {
		return domain.InventoryDocument{}, err
	}
	return posted, nil
}

// topicForDocument picks the topic a movement is published under. A shipment
// and a reversal get their own, because a consumer that only cares about goods
// leaving the gate should not have to parse every stock posting to find them.
func topicForDocument(t domain.DocType) domain.Topic {
	switch t {
	case domain.DocShipment:
		return domain.TopicShipmentDispatched
	case domain.DocReversal:
		return domain.TopicStockReversed
	}
	return domain.TopicStockPosted
}

func yearOf(d domain.BusinessDate) (int, error) {
	t, err := d.Time()
	if err != nil {
		return 0, fmt.Errorf("%w: %q is not a valid business date", domain.ErrValidation, d)
	}
	return t.Year(), nil
}

// ---------------------------------------------------------------------------
// Reversal
// ---------------------------------------------------------------------------

// ReversalRequest asks for a posted document to be undone.
type ReversalRequest struct {
	BusinessDate domain.BusinessDate `json:"businessDate,omitempty"`
	Reason       string              `json:"reason"`
}

// Reverse posts the counter-document of an existing posting.
//
// Nothing is deleted and nothing is edited: both documents stay in the ledger,
// which is what lets somebody six months later see that a mistake was made and
// what was done about it.
func (e *Execution) Reverse(ctx context.Context, documentID string, req ReversalRequest) (domain.InventoryDocument, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermActualStock); err != nil {
		return domain.InventoryDocument{}, err
	}

	original, err := e.store.Execution().GetDocument(ctx, documentID)
	if err != nil {
		return domain.InventoryDocument{}, err
	}
	if err := caller.RequireFactory(original.FactoryID); err != nil {
		return domain.InventoryDocument{}, err
	}

	reversal, err := domain.ReverseDocument(original, req.BusinessDate, req.Reason, caller.Username)
	if err != nil {
		return domain.InventoryDocument{}, err
	}

	// A reversal restores the position the original moved away from, and the
	// stock it brought in may well have moved on since. Refusing it because the
	// balance would go negative would leave a wrong posting standing, which is
	// worse than a balance that shows the truth about a shed somebody has
	// already emptied.
	opts := domain.PostingOptions{AllowNegativeStock: true, CapacityOverride: true}

	var posted domain.InventoryDocument
	err = e.store.InTx(ctx, func(tx store.Store) error {
		var err error
		posted, err = e.postChecked(ctx, tx, reversal, opts)
		if err != nil {
			return err
		}
		if err := tx.Execution().MarkReversed(ctx, original.ID, caller.Username); err != nil {
			return err
		}
		return e.audit(ctx, tx, auditEntry{
			action: "REVERSE", entity: "inventory_document", entityID: original.ID,
			before: original, after: posted, reason: req.Reason,
		})
	})
	if err != nil {
		return domain.InventoryDocument{}, err
	}
	return posted, nil
}

// ---------------------------------------------------------------------------
// Reads
// ---------------------------------------------------------------------------

// ListDocuments returns the postings the caller may see.
func (e *Execution) ListDocuments(ctx context.Context, f store.ExecutionFilter) (store.Page[domain.InventoryDocument], error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return store.Page[domain.InventoryDocument]{}, err
	}
	if err := e.requireScope(caller, f.FactoryID); err != nil {
		return store.Page[domain.InventoryDocument]{}, err
	}
	return e.store.Execution().ListDocuments(ctx, f)
}

// GetDocument reads one posting.
func (e *Execution) GetDocument(ctx context.Context, id string) (domain.InventoryDocument, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return domain.InventoryDocument{}, err
	}
	d, err := e.store.Execution().GetDocument(ctx, id)
	if err != nil {
		return domain.InventoryDocument{}, err
	}
	if err := caller.RequireFactory(d.FactoryID); err != nil {
		return domain.InventoryDocument{}, err
	}
	return d, nil
}

// StockLine is one warehouse and product position, with the master data a
// screen needs to make sense of it.
type StockLine struct {
	WarehouseID   string     `json:"warehouseId"`
	WarehouseCode string     `json:"warehouseCode"`
	WarehouseName string     `json:"warehouseName"`
	ProductID     string     `json:"productId"`
	ProductCode   string     `json:"productCode"`
	ProductName   string     `json:"productName"`
	Quantity      domain.Dec `json:"quantity"`
	HoldQuantity  domain.Dec `json:"holdQuantity"`
	Available     domain.Dec `json:"available"`
	Capacity      domain.Dec `json:"capacity"`
	UtilisedPct   domain.Dec `json:"utilisedPct"`
	UOM           string     `json:"uom"`
}

// Stock returns the current position of every product in the warehouses the
// caller may see.
func (e *Execution) Stock(ctx context.Context, f store.ExecutionFilter) ([]StockLine, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return nil, err
	}
	positions, err := e.store.Execution().ListPositions(ctx, f)
	if err != nil {
		return nil, err
	}

	md := e.store.MasterData()
	warehouses := map[string]domain.Warehouse{}
	products := map[string]domain.Product{}

	out := make([]StockLine, 0, len(positions))
	for _, p := range positions {
		warehouse, ok := warehouses[p.WarehouseID]
		if !ok {
			if warehouse, err = md.Warehouses().Get(ctx, p.WarehouseID); err != nil {
				return nil, err
			}
			warehouses[p.WarehouseID] = warehouse
		}
		// The data scope is applied here rather than in the repository, so that
		// a repository can never be the thing that leaks another factory's
		// stock.
		if !caller.CanSeeFactory(warehouse.FactoryID) {
			continue
		}
		product, ok := products[p.ProductID]
		if !ok {
			if product, err = md.Products().Get(ctx, p.ProductID); err != nil {
				return nil, err
			}
			products[p.ProductID] = product
		}

		capacity := warehouse.UsableCapacity()
		out = append(out, StockLine{
			WarehouseID: p.WarehouseID, WarehouseCode: warehouse.Code, WarehouseName: warehouse.Name,
			ProductID: p.ProductID, ProductCode: product.Code, ProductName: product.Name,
			Quantity: domain.RoundQty(p.Quantity), HoldQuantity: domain.RoundQty(p.HoldQuantity),
			Available: p.Available(), Capacity: capacity,
			UtilisedPct: domain.RoundPct(domain.SafePct(p.Quantity, capacity)),
			UOM:         product.BaseUOM,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].WarehouseCode != out[j].WarehouseCode {
			return out[i].WarehouseCode < out[j].WarehouseCode
		}
		return out[i].ProductCode < out[j].ProductCode
	})
	return out, nil
}

// requireScope checks a filter that names a factory, and refuses a filter that
// names none from a caller who is not allowed to see everything.
func (e *Execution) requireScope(caller auth.Principal, factoryID string) error {
	if factoryID != "" {
		return caller.RequireFactory(factoryID)
	}
	if len(caller.Factories) == 0 && len(caller.Companies) == 0 {
		return fmt.Errorf("%w: %s has no factory in scope", domain.ErrForbidden, caller.Username)
	}
	return nil
}
