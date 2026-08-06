package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Integration events are written where the change is made, inside the same
// transaction, through this one helper.
//
// Publishing from the service after the transaction commits would be simpler to
// read and wrong in a way that only shows up on a bad night: the process can die
// between the commit and the call, and the ERP would never hear about a
// confirmation that this system considers posted. Writing the event with the
// change makes the two atomic, and a separate dispatcher takes it from there.

// emit writes an integration event alongside the change that caused it.
//
// It takes the caller's transaction rather than the service's store, which is
// what makes the guarantee: if the surrounding transaction rolls back, so does
// the event.
func emit(ctx context.Context, tx store.Store, topic domain.Topic, payload any) error {
	if !domain.ValidTopic(topic) {
		return fmt.Errorf("%w: %q is not a topic this system publishes", domain.ErrValidation, topic)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", topic, err)
	}
	return tx.Outbox().Append(ctx, domain.OutboxEvent{
		Topic:         topic,
		Payload:       string(body),
		CorrelationID: CorrelationFromContext(ctx),
	})
}

// The payloads below are the published contract. They are deliberately separate
// from the internal entities: a consumer should not be broken by a column being
// added here, and this system should not be prevented from adding one because a
// consumer parses the whole row. Each carries the business keys a consumer needs
// to find the same thing in its own system - document numbers, product and
// warehouse codes - rather than only this system's identifiers.

// StockEvent describes a posting that moved stock.
type StockEvent struct {
	DocumentID   string              `json:"documentId"`
	DocumentNo   string              `json:"documentNo"`
	DocType      domain.DocType      `json:"docType"`
	BusinessDate domain.BusinessDate `json:"businessDate"`
	FactoryID    string              `json:"factoryId"`
	Reference    string              `json:"reference,omitempty"`
	ReversalOf   string              `json:"reversalOf,omitempty"`
	ReasonCode   string              `json:"reasonCode,omitempty"`
	PostedBy     string              `json:"postedBy"`
	Lines        []StockEventLine    `json:"lines"`
}

// StockEventLine is one movement, signed the way the ledger holds it.
type StockEventLine struct {
	LineNo      int        `json:"lineNo"`
	WarehouseID string     `json:"warehouseId"`
	ProductID   string     `json:"productId"`
	BatchID     string     `json:"batchId,omitempty"`
	Quantity    domain.Dec `json:"quantity"`
	UOM         string     `json:"uom,omitempty"`
}

func stockEvent(d domain.InventoryDocument) StockEvent {
	lines := make([]StockEventLine, 0, len(d.Items))
	for _, item := range d.Items {
		lines = append(lines, StockEventLine{
			LineNo: item.LineNo, WarehouseID: item.WarehouseID, ProductID: item.ProductID,
			BatchID: item.BatchID, Quantity: item.Quantity, UOM: item.UOM,
		})
	}
	return StockEvent{
		DocumentID: d.ID, DocumentNo: d.DocumentNo, DocType: d.DocType,
		BusinessDate: d.BusinessDate, FactoryID: d.FactoryID, Reference: d.Reference,
		ReversalOf: d.ReversalOf, ReasonCode: d.ReasonCode, PostedBy: d.CreatedBy, Lines: lines,
	}
}

// ConfirmationEvent describes production confirmed against an order.
type ConfirmationEvent struct {
	ConfirmationID string              `json:"confirmationId"`
	ConfirmationNo string              `json:"confirmationNo"`
	OrderID        string              `json:"orderId"`
	OrderNo        string              `json:"orderNo"`
	FactoryID      string              `json:"factoryId"`
	BusinessDate   domain.BusinessDate `json:"businessDate"`
	ProductID      string              `json:"productId"`
	BatchID        string              `json:"batchId,omitempty"`
	GoodQuantity   domain.Dec          `json:"goodQuantity"`
	ScrapQuantity  domain.Dec          `json:"scrapQuantity"`
	ReworkQuantity domain.Dec          `json:"reworkQuantity"`
	OrderStatus    domain.OrderStatus  `json:"orderStatus"`
	Final          bool                `json:"final"`
	ConfirmedBy    string              `json:"confirmedBy"`
}

// QualityEvent describes a verdict or a release. A consumer that blocks its own
// stock on a failure needs the material and the quantity, not the readings.
type QualityEvent struct {
	SampleID     string               `json:"sampleId,omitempty"`
	SampleNo     string               `json:"sampleNo,omitempty"`
	HoldID       string               `json:"holdId,omitempty"`
	FactoryID    string               `json:"factoryId"`
	BusinessDate domain.BusinessDate  `json:"businessDate"`
	ProductID    string               `json:"productId"`
	BatchID      string               `json:"batchId,omitempty"`
	WarehouseID  string               `json:"warehouseId,omitempty"`
	Quantity     domain.Dec           `json:"quantity,omitempty"`
	Verdict      domain.QualityStatus `json:"verdict,omitempty"`
	Reason       string               `json:"reason,omitempty"`
	Actor        string               `json:"actor"`
}

// PlanReleasedEvent tells the receiving systems that a plan is now the one to
// work to. The figures are not repeated here: a consumer that wants the daily
// rows reads them from the API, and duplicating a season's numbers into a
// message queue would guarantee the two drift apart.
type PlanReleasedEvent struct {
	VersionID   string              `json:"versionId"`
	VersionNo   int                 `json:"versionNo"`
	SeasonID    string              `json:"seasonId"`
	SeasonCode  string              `json:"seasonCode,omitempty"`
	FactoryID   string              `json:"factoryId"`
	PlanType    domain.PlanType     `json:"planType"`
	Description string              `json:"description,omitempty"`
	ReleasedBy  string              `json:"releasedBy"`
	ValidFrom   domain.BusinessDate `json:"validFrom,omitempty"`
	ValidTo     domain.BusinessDate `json:"validTo,omitempty"`
}

// CostRunEvent tells the finance system that a costing has been saved.
type CostRunEvent struct {
	RunID       string              `json:"runId"`
	SeasonID    string              `json:"seasonId"`
	VersionID   string              `json:"versionId,omitempty"`
	FactoryID   string              `json:"factoryId"`
	From        domain.BusinessDate `json:"from"`
	To          domain.BusinessDate `json:"to"`
	Currency    string              `json:"currency"`
	PlannedCost domain.Dec          `json:"plannedCost"`
	ActualCost  domain.Dec          `json:"actualCost"`
	Variance    domain.Dec          `json:"variance"`
	RunBy       string              `json:"runBy"`
}
