package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// These tests exercise the daily factory: postings, orders, confirmations and
// the laboratory. They run against the seeded reference scenario, so the
// quantities and the warehouse capacities are the real ones from section 2.

func execHarness(t *testing.T) (*harness, *service.Execution) {
	t.Helper()
	h := newHarness(t, 0)
	return h, service.NewExecution(h.store, fixedClock())
}

// keeper is a warehouse operator, which is the role that posts stock.
func (h *harness) keeper() context.Context { return h.as(auth.RoleWarehouseOperator) }

// ---------------------------------------------------------------------------
// Inventory postings
// ---------------------------------------------------------------------------

func TestReceiptAndIssueMoveTheBalance(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.keeper()
	warehouse := h.seeded.Warehouses["FG-WH1"]
	product := h.seeded.Products["REF"]

	receipt, err := exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: warehouse, ProductID: product, Quantity: domain.D("500"),
		}},
	})
	if err != nil {
		t.Fatalf("post receipt: %v", err)
	}
	if !strings.HasPrefix(receipt.DocumentNo, "MD-F1-2026-") {
		t.Errorf("document number = %q, want the MD series for the factory and year", receipt.DocumentNo)
	}
	if receipt.Items[0].Quantity.LessThanOrEqual(domain.Zero) {
		t.Errorf("a receipt line is positive, got %s", receipt.Items[0].Quantity)
	}

	// An issue is entered as a positive number and stored with its sign, which
	// is the ledger's job, not the keeper's.
	issue, err := exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocIssue, BusinessDate: "2026-12-06", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: warehouse, ProductID: product, Quantity: domain.D("120"),
		}},
	})
	if err != nil {
		t.Fatalf("post issue: %v", err)
	}
	if !issue.Items[0].Quantity.Equal(domain.D("-120")) {
		t.Errorf("an issue line is negative, got %s", issue.Items[0].Quantity)
	}

	stock := stockFor(t, ctx, exec, warehouse, product)
	if !stock.Quantity.Equal(domain.D("380")) {
		t.Errorf("balance = %s, want 380", stock.Quantity)
	}
	if !stock.Available.Equal(domain.D("380")) {
		t.Errorf("available = %s, want 380 with nothing on hold", stock.Available)
	}
}

func TestAnIssueBeyondTheBalanceIsRefusedWithTheFigures(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.keeper()
	warehouse := h.seeded.Warehouses["FG-WH1"]
	product := h.seeded.Products["REF"]

	if _, err := exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: warehouse, ProductID: product, Quantity: domain.D("100"),
		}},
	}); err != nil {
		t.Fatalf("post receipt: %v", err)
	}

	_, err := exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocIssue, BusinessDate: "2026-12-06", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: warehouse, ProductID: product, Quantity: domain.D("150"),
		}},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("issuing more than is there must be refused, got %v", err)
	}
	var verr *domain.ValidationError
	if !errors.As(err, &verr) || len(verr.Errors) == 0 {
		t.Fatalf("the refusal must name the offending line, got %v", err)
	}
	if verr.Errors[0].Code != "NEGATIVE_STOCK" {
		t.Errorf("code = %q, want NEGATIVE_STOCK", verr.Errors[0].Code)
	}
	if verr.Errors[0].Row == nil || *verr.Errors[0].Row != 1 {
		t.Errorf("the error must address line 1, got %+v", verr.Errors[0])
	}
	if !strings.Contains(verr.Errors[0].Message, "100") {
		t.Errorf("the message must say what is actually there: %q", verr.Errors[0].Message)
	}

	// Nothing was written: a refused posting leaves no document behind.
	documents, err := exec.ListDocuments(ctx, store.ExecutionFilter{FactoryID: h.seeded.FactoryID})
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if documents.Count != 1 {
		t.Errorf("documents = %d, want only the receipt", documents.Count)
	}
}

func TestAReceiptBeyondCapacityNeedsAnAuthorisedOverride(t *testing.T) {
	h, exec := execHarness(t)
	warehouse := h.seeded.Warehouses["FG-WH1"] // 22,000 t of usable capacity
	product := h.seeded.Products["REF"]

	request := service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: warehouse, ProductID: product, Quantity: domain.D("22500"),
		}},
	}

	_, err := exec.Post(h.keeper(), request)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("a receipt past capacity must be refused, got %v", err)
	}
	if !strings.Contains(err.Error(), "22000") {
		t.Errorf("the refusal must quote the usable capacity: %v", err)
	}

	// The keeper may not authorise their own override.
	request.CapacityOverride = true
	if _, err := exec.Post(h.keeper(), request); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("a keeper cannot override capacity, got %v", err)
	}

	// An approver may.
	approver := h.as(auth.RoleApprover, auth.RoleWarehouseOperator)
	posted, err := exec.Post(approver, request)
	if err != nil {
		t.Fatalf("post with an override: %v", err)
	}
	if posted.ID == "" {
		t.Fatal("the override posting produced no document")
	}

	events, err := h.store.Audit().List(context.Background(), store.AuditFilter{
		Entity: "inventory_document", EntityID: posted.ID,
	})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if events.Count != 1 || !strings.Contains(events.Items[0].Reason, "capacity override") {
		t.Errorf("the override must be recorded on the audit event, got %+v", events.Items)
	}
}

func TestATransferMovesBothBalancesOrNeither(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.keeper()
	from, to := h.seeded.Warehouses["FG-WH1"], h.seeded.Warehouses["FG-WH3"]
	product := h.seeded.Products["REF"]

	if _, err := exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: from, ProductID: product, Quantity: domain.D("1000"),
		}},
	}); err != nil {
		t.Fatalf("post receipt: %v", err)
	}

	transfer, err := exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocTransfer, BusinessDate: "2026-12-06", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: from, ToWarehouse: to, ProductID: product, Quantity: domain.D("400"),
		}},
	})
	if err != nil {
		t.Fatalf("post transfer: %v", err)
	}
	// One request line becomes two ledger lines, so the halves cannot be
	// posted apart from one another.
	if len(transfer.Items) != 2 {
		t.Fatalf("a transfer has two lines, got %d", len(transfer.Items))
	}
	if !transfer.Items[0].Quantity.Add(transfer.Items[1].Quantity).IsZero() {
		t.Error("the two halves of a transfer must cancel out")
	}

	if got := stockFor(t, ctx, exec, from, product).Quantity; !got.Equal(domain.D("600")) {
		t.Errorf("source balance = %s, want 600", got)
	}
	if got := stockFor(t, ctx, exec, to, product).Quantity; !got.Equal(domain.D("400")) {
		t.Errorf("target balance = %s, want 400", got)
	}

	// A transfer that would empty the source past zero moves nothing at all.
	_, err = exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocTransfer, BusinessDate: "2026-12-07", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: from, ToWarehouse: to, ProductID: product, Quantity: domain.D("900"),
		}},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("an oversized transfer must be refused, got %v", err)
	}
	if got := stockFor(t, ctx, exec, to, product).Quantity; !got.Equal(domain.D("400")) {
		t.Errorf("the target balance moved despite the refusal: %s", got)
	}
}

func TestReversingAPostingLeavesBothDocumentsInTheLedger(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.keeper()
	warehouse := h.seeded.Warehouses["FG-WH1"]
	product := h.seeded.Products["REF"]

	receipt, err := exec.Post(ctx, service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: warehouse, ProductID: product, Quantity: domain.D("500"),
		}},
	})
	if err != nil {
		t.Fatalf("post receipt: %v", err)
	}

	if _, err := exec.Reverse(ctx, receipt.ID, service.ReversalRequest{}); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("a reversal without a reason must be refused, got %v", err)
	}

	reversal, err := exec.Reverse(ctx, receipt.ID, service.ReversalRequest{
		BusinessDate: "2026-12-07", Reason: "posted to the wrong store",
	})
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if reversal.ReversalOf != receipt.ID || reversal.DocType != domain.DocReversal {
		t.Errorf("the reversal must point back at the original: %+v", reversal)
	}
	if got := stockFor(t, ctx, exec, warehouse, product).Quantity; !got.IsZero() {
		t.Errorf("balance after the reversal = %s, want 0", got)
	}

	original, err := exec.GetDocument(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("get original: %v", err)
	}
	if !original.Reversed {
		t.Error("the original must be flagged, not deleted")
	}
	if _, err := exec.Reverse(ctx, receipt.ID, service.ReversalRequest{Reason: "again"}); err == nil {
		t.Error("a document may only be reversed once")
	}

	documents, err := exec.ListDocuments(ctx, store.ExecutionFilter{FactoryID: h.seeded.FactoryID})
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if documents.Count != 2 {
		t.Errorf("both documents stay in the ledger, got %d", documents.Count)
	}
}

func TestPostingRefusesACallerWithoutTheStockPermission(t *testing.T) {
	h, exec := execHarness(t)
	planner := h.as(auth.RoleProductionPlanner)

	_, err := exec.Post(planner, service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: h.seeded.Warehouses["FG-WH1"],
			ProductID:   h.seeded.Products["REF"], Quantity: domain.D("1"),
		}},
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a planner may not post stock, got %v", err)
	}
}

func TestPostingRefusesAFactoryOutOfScope(t *testing.T) {
	h, exec := execHarness(t)
	outsider := auth.WithPrincipal(context.Background(), auth.NewPrincipal(
		"other", "other-keeper", "Other keeper", "",
		[]string{auth.RoleWarehouseOperator}, nil, []string{"11111111-1111-4111-8111-111111111111"}))

	_, err := exec.Post(outsider, service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: h.seeded.Warehouses["FG-WH1"],
			ProductID:   h.seeded.Products["REF"], Quantity: domain.D("1"),
		}},
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("another factory's keeper may not post here, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Production orders
// ---------------------------------------------------------------------------

func TestOrdersAreCreatedFromTheReleasedPlanAndNotDuplicated(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.as(auth.RoleShiftSupervisor)

	// The seeded budget version is not released, so it may not become orders.
	_, err := exec.CreateOrdersFromPlan(ctx, h.seeded.BudgetID, service.OrdersFromPlanRequest{
		From: "2026-12-01", To: "2026-12-03",
	})
	if !errors.Is(err, domain.ErrStateTransition) {
		t.Fatalf("only a released plan may become orders, got %v", err)
	}

	released := releaseSeededPlan(t, h)

	result, err := exec.CreateOrdersFromPlan(ctx, released, service.OrdersFromPlanRequest{
		From: "2026-12-01", To: "2026-12-03",
	})
	if err != nil {
		t.Fatalf("create orders from plan: %v", err)
	}
	if len(result.Created) == 0 {
		t.Fatal("the released plan produced no orders")
	}
	for _, o := range result.Created {
		if o.Status != domain.OrderPlanned {
			t.Errorf("a new order starts PLANNED, got %s", o.Status)
		}
		if o.VersionID != released {
			t.Errorf("the order must remember the plan it came from, got %q", o.VersionID)
		}
		if o.PlannedQty.LessThanOrEqual(domain.Zero) {
			t.Errorf("order %s has nothing to make", o.OrderNo)
		}
	}

	// Running it again over the same range creates nothing and says why.
	again, err := exec.CreateOrdersFromPlan(ctx, released, service.OrdersFromPlanRequest{
		From: "2026-12-01", To: "2026-12-03",
	})
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(again.Created) != 0 {
		t.Errorf("the second run created %d orders; it must be repeatable", len(again.Created))
	}
	if len(again.Skipped) != len(result.Created) {
		t.Errorf("skipped = %d, want the %d already covered", len(again.Skipped), len(result.Created))
	}
	if len(again.Skipped) > 0 && !strings.Contains(again.Skipped[0].Reason, "already covers") {
		t.Errorf("the skip must explain itself: %q", again.Skipped[0].Reason)
	}
}

func TestTheOrderLifeCycleRefusesStepsOutOfOrder(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.as(auth.RoleShiftSupervisor)
	order := seedOrder(t, h, exec, ctx, domain.D("300"))

	// A planned order cannot be confirmed before it is released.
	if _, err := exec.Confirm(ctx, order.ID, service.ConfirmRequest{
		YieldQty: domain.D("10"),
	}); !errors.Is(err, domain.ErrStateTransition) {
		t.Fatalf("a planned order may not be confirmed, got %v", err)
	}

	released, err := exec.ActOnOrder(ctx, order.ID, service.OrderActionRequest{
		Action: domain.OrderActionRelease, RowVersion: order.RowVersion,
	})
	if err != nil {
		t.Fatalf("release order: %v", err)
	}
	if released.Status != domain.OrderReleased {
		t.Fatalf("status after release = %s", released.Status)
	}

	// The same command twice is a state error, not a silent no-op.
	if _, err := exec.ActOnOrder(ctx, order.ID, service.OrderActionRequest{
		Action: domain.OrderActionRelease,
	}); !errors.Is(err, domain.ErrStateTransition) {
		t.Errorf("releasing twice must be refused, got %v", err)
	}

	// A stale row version is a conflict.
	if _, err := exec.ActOnOrder(ctx, order.ID, service.OrderActionRequest{
		Action: domain.OrderActionCancel, RowVersion: order.RowVersion,
	}); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("a stale version must conflict, got %v", err)
	}
}

func TestConfirmingReceiptsTheYieldAndAdvancesTheOrder(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.as(auth.RoleShiftSupervisor)
	warehouse := h.seeded.Warehouses["FG-WH1"]
	product := h.seeded.Products["REF"]

	order := seedOrder(t, h, exec, ctx, domain.D("300"))
	order, err := exec.ActOnOrder(ctx, order.ID, service.OrderActionRequest{
		Action: domain.OrderActionRelease, RowVersion: order.RowVersion,
	})
	if err != nil {
		t.Fatalf("release: %v", err)
	}

	first, err := exec.Confirm(ctx, order.ID, service.ConfirmRequest{
		BusinessDate: "2026-12-05", YieldQty: domain.D("180"), ScrapQty: domain.D("2"),
		WarehouseID: warehouse,
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if first.Order.Status != domain.OrderPartiallyConfirmed {
		t.Errorf("status after a part confirmation = %s", first.Order.Status)
	}
	if !first.Order.OpenQty().Equal(domain.D("120")) {
		t.Errorf("open quantity = %s, want 120", first.Order.OpenQty())
	}
	if first.Document == nil {
		t.Fatal("the yield must be receipted into stock")
	}
	// Only the good output goes into the shed; the scrap does not.
	if got := stockFor(t, h.keeper(), exec, warehouse, product).Quantity; !got.Equal(domain.D("180")) {
		t.Errorf("balance after the confirmation = %s, want the yield of 180", got)
	}

	second, err := exec.Confirm(ctx, order.ID, service.ConfirmRequest{
		BusinessDate: "2026-12-06", YieldQty: domain.D("125"), WarehouseID: warehouse,
	})
	if err != nil {
		t.Fatalf("second confirm: %v", err)
	}
	// Factories make more than planned; over-confirmation completes the order.
	if second.Order.Status != domain.OrderCompleted {
		t.Errorf("status after meeting the plan = %s, want COMPLETED", second.Order.Status)
	}
	if !second.Order.ConfirmedQty.Equal(domain.D("305")) {
		t.Errorf("confirmed = %s, want 305", second.Order.ConfirmedQty)
	}

	detail, err := exec.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if len(detail.Confirmations) != 2 {
		t.Errorf("confirmations = %d, want 2", len(detail.Confirmations))
	}
	if !detail.VariancePct.Equal(domain.D("1.667")) {
		t.Errorf("variance = %s%%, want 1.667", detail.VariancePct)
	}
}

func TestReversingAConfirmationUndoesBothTheOrderAndTheStock(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.as(auth.RoleShiftSupervisor)
	warehouse := h.seeded.Warehouses["FG-WH1"]
	product := h.seeded.Products["REF"]

	order := seedOrder(t, h, exec, ctx, domain.D("300"))
	order, err := exec.ActOnOrder(ctx, order.ID, service.OrderActionRequest{
		Action: domain.OrderActionRelease, RowVersion: order.RowVersion,
	})
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	confirmed, err := exec.Confirm(ctx, order.ID, service.ConfirmRequest{
		BusinessDate: "2026-12-05", YieldQty: domain.D("180"), WarehouseID: warehouse,
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}

	reversed, err := exec.ReverseConfirmation(ctx, confirmed.Confirmation.ID, service.ReversalRequest{
		Reason: "the weighbridge ticket was misread",
	})
	if err != nil {
		t.Fatalf("reverse confirmation: %v", err)
	}
	if !reversed.Confirmation.Reversed {
		t.Error("the confirmation must be flagged as reversed")
	}
	if !reversed.Order.ConfirmedQty.IsZero() {
		t.Errorf("confirmed quantity after the reversal = %s, want 0", reversed.Order.ConfirmedQty)
	}
	if reversed.Order.Status != domain.OrderReleased {
		t.Errorf("the order returns to RELEASED, got %s", reversed.Order.Status)
	}
	if got := stockFor(t, h.keeper(), exec, warehouse, product).Quantity; !got.IsZero() {
		t.Errorf("balance after the reversal = %s, want 0", got)
	}
	if _, err := exec.ReverseConfirmation(ctx, confirmed.Confirmation.ID, service.ReversalRequest{
		Reason: "again",
	}); err == nil {
		t.Error("a confirmation may only be reversed once")
	}
}

func TestClosingAnOrderOutsideToleranceNeedsAnAuthorisedReason(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.as(auth.RoleShiftSupervisor)

	order := seedOrder(t, h, exec, ctx, domain.D("300"))
	order, err := exec.ActOnOrder(ctx, order.ID, service.OrderActionRequest{
		Action: domain.OrderActionRelease, RowVersion: order.RowVersion,
	})
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	// 200 t against a plan of 300 t is a third short, well outside the 5%
	// default tolerance.
	if _, err := exec.Confirm(ctx, order.ID, service.ConfirmRequest{
		BusinessDate: "2026-12-05", YieldQty: domain.D("200"),
	}); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	_, err = exec.ActOnOrder(ctx, order.ID, service.OrderActionRequest{
		Action: domain.OrderActionClose,
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("closing short without a reason must be refused, got %v", err)
	}
	if !strings.Contains(err.Error(), "-33.333") {
		t.Errorf("the refusal must quote the variance: %v", err)
	}

	// A reason is not enough on its own; the closer has to be an approver.
	_, err = exec.ActOnOrder(ctx, order.ID, service.OrderActionRequest{
		Action: domain.OrderActionClose, VarianceReason: "VAR-CANE",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("a supervisor may not authorise their own variance, got %v", err)
	}

	approver := h.as(auth.RoleApprover, auth.RoleShiftSupervisor)
	closed, err := exec.ActOnOrder(approver, order.ID, service.OrderActionRequest{
		Action: domain.OrderActionClose, VarianceReason: "VAR-CANE",
		Reason: "cane quality below specification all week",
	})
	if err != nil {
		t.Fatalf("close with an authorised variance: %v", err)
	}
	if closed.Status != domain.OrderTechnicallyClosed {
		t.Errorf("status = %s, want TECHNICALLY_CLOSED", closed.Status)
	}
	if closed.VarianceReason != "VAR-CANE" {
		t.Errorf("the reason must stay on the order, got %q", closed.VarianceReason)
	}
}

// ---------------------------------------------------------------------------
// Quality
// ---------------------------------------------------------------------------

func TestAFailedSampleBlocksTheStockItCovers(t *testing.T) {
	h, exec := execHarness(t)
	admin := h.as(auth.RoleMasterDataAdmin)
	lab := h.as(auth.RoleQualityUser)
	keeper := h.keeper()
	warehouse := h.seeded.Warehouses["FG-WH1"]
	product := h.seeded.Products["REF"]

	if _, err := exec.Post(keeper, service.PostingRequest{
		DocType: domain.DocReceipt, BusinessDate: "2026-12-05", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: warehouse, ProductID: product, Quantity: domain.D("500"),
		}},
	}); err != nil {
		t.Fatalf("post receipt: %v", err)
	}

	pol, err := exec.SaveParameter(admin, domain.QualityParameter{
		Code: "POL", Name: "Polarisation", UOM: "PCT", Validity: domain.Validity{Active: true},
	})
	if err != nil {
		t.Fatalf("save parameter: %v", err)
	}
	lower := domain.D("99.500")
	if _, err := exec.SaveSpec(admin, domain.QualitySpec{
		ProductID: product, ParameterID: pol.ID, LowerLimit: &lower, ValidFrom: "2026-10-01",
	}); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	sample, err := exec.CreateSample(lab, service.SampleRequest{
		ProductID: product, FactoryID: h.seeded.FactoryID, BusinessDate: "2026-12-06",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}
	if !strings.HasPrefix(sample.SampleNo, "QS-F1-2026-") {
		t.Errorf("sample number = %q, want the QS series", sample.SampleNo)
	}

	outcome, err := exec.RecordResults(lab, sample.ID, service.ResultsRequest{
		Results:  []service.ResultInput{{ParameterID: pol.ID, Value: domain.D("98.900")}},
		Complete: true, HoldWarehouse: warehouse, HoldQuantity: domain.D("120"),
	})
	if err != nil {
		t.Fatalf("record results: %v", err)
	}
	if outcome.Verdict != domain.QualityFail {
		t.Fatalf("98.9 is below the limit of 99.5, verdict = %s", outcome.Verdict)
	}
	if outcome.Hold == nil {
		t.Fatal("a failed sample must block the material")
	}
	// The result keeps the limit it was judged against, so editing the
	// specification later cannot rewrite history.
	if len(outcome.Sample.Results) != 1 || outcome.Sample.Results[0].LowerLimit == nil ||
		!outcome.Sample.Results[0].LowerLimit.Equal(lower) {
		t.Errorf("the result must carry its limits: %+v", outcome.Sample.Results)
	}

	stock := stockFor(t, keeper, exec, warehouse, product)
	if !stock.HoldQuantity.Equal(domain.D("120")) {
		t.Errorf("held = %s, want 120", stock.HoldQuantity)
	}
	if !stock.Available.Equal(domain.D("380")) {
		t.Errorf("available = %s, want 380 of the 500 on hand", stock.Available)
	}

	// Held stock may not be issued, even though it is physically there.
	_, err = exec.Post(keeper, service.PostingRequest{
		DocType: domain.DocIssue, BusinessDate: "2026-12-07", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: warehouse, ProductID: product, Quantity: domain.D("450"),
		}},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("shipping held stock must be refused, got %v", err)
	}
	if !strings.Contains(err.Error(), "quality hold") {
		t.Errorf("the refusal must say the stock is held: %v", err)
	}

	// Releasing the hold frees it again.
	if _, err := exec.ReleaseHold(h.as(auth.RoleShiftSupervisor), outcome.Hold.ID,
		service.ReleaseRequest{ReleasedOn: "2026-12-08", Reason: "retested within limits"},
	); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a supervisor may not release a quality hold, got %v", err)
	}

	released, err := exec.ReleaseHold(lab, outcome.Hold.ID, service.ReleaseRequest{
		ReleasedOn: "2026-12-08", Reason: "retested within limits",
		RowVersion: outcome.Hold.RowVersion,
	})
	if err != nil {
		t.Fatalf("release hold: %v", err)
	}
	if released.IsOpen() {
		t.Error("the hold must be closed")
	}
	if got := stockFor(t, keeper, exec, warehouse, product); !got.HoldQuantity.IsZero() {
		t.Errorf("held after the release = %s, want 0", got.HoldQuantity)
	}
	if _, err := exec.Post(keeper, service.PostingRequest{
		DocType: domain.DocIssue, BusinessDate: "2026-12-09", FactoryID: h.seeded.FactoryID,
		Lines: []service.PostingLineInput{{
			WarehouseID: warehouse, ProductID: product, Quantity: domain.D("450"),
		}},
	}); err != nil {
		t.Errorf("the released stock must be issuable: %v", err)
	}
}

func TestAMeasurementWithNoSpecificationIsReportedRatherThanPassedQuietly(t *testing.T) {
	h, exec := execHarness(t)
	admin := h.as(auth.RoleMasterDataAdmin)
	lab := h.as(auth.RoleQualityUser)
	product := h.seeded.Products["REF"]

	colour, err := exec.SaveParameter(admin, domain.QualityParameter{
		Code: "COLOUR", Name: "Colour", UOM: "IU", Validity: domain.Validity{Active: true},
	})
	if err != nil {
		t.Fatalf("save parameter: %v", err)
	}
	sample, err := exec.CreateSample(lab, service.SampleRequest{
		ProductID: product, FactoryID: h.seeded.FactoryID, BusinessDate: "2026-12-06",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}

	outcome, err := exec.RecordResults(lab, sample.ID, service.ResultsRequest{
		Results:  []service.ResultInput{{ParameterID: colour.ID, Value: domain.D("450")}},
		Complete: true,
	})
	if err != nil {
		t.Fatalf("record results: %v", err)
	}
	if outcome.Verdict != domain.QualityPass {
		t.Errorf("with no limits nothing can fail, verdict = %s", outcome.Verdict)
	}
	if len(outcome.Unspecified) != 1 || outcome.Unspecified[0] != "COLOUR" {
		t.Errorf("the configuration gap must be reported, got %+v", outcome.Unspecified)
	}
}

func TestASampleTakesItsResultsOnlyOnce(t *testing.T) {
	h, exec := execHarness(t)
	admin := h.as(auth.RoleMasterDataAdmin)
	lab := h.as(auth.RoleQualityUser)

	pol, err := exec.SaveParameter(admin, domain.QualityParameter{
		Code: "POL", Name: "Polarisation", Validity: domain.Validity{Active: true},
	})
	if err != nil {
		t.Fatalf("save parameter: %v", err)
	}
	sample, err := exec.CreateSample(lab, service.SampleRequest{
		ProductID: h.seeded.Products["REF"], FactoryID: h.seeded.FactoryID,
		BusinessDate: "2026-12-06",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}

	if _, err := exec.RecordResults(lab, sample.ID, service.ResultsRequest{
		Results: []service.ResultInput{
			{ParameterID: pol.ID, Value: domain.D("99.9")},
			{ParameterID: pol.ID, Value: domain.D("99.8")},
		},
	}); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("the same parameter twice on one sheet must be refused, got %v", err)
	}

	if _, err := exec.RecordResults(lab, sample.ID, service.ResultsRequest{
		Results:  []service.ResultInput{{ParameterID: pol.ID, Value: domain.D("99.9")}},
		Complete: true,
	}); err != nil {
		t.Fatalf("record results: %v", err)
	}
	if _, err := exec.RecordResults(lab, sample.ID, service.ResultsRequest{
		Results: []service.ResultInput{{ParameterID: pol.ID, Value: domain.D("99.1")}},
	}); !errors.Is(err, domain.ErrStateTransition) {
		t.Errorf("a completed sample takes no more results, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Maintenance
// ---------------------------------------------------------------------------

func TestApprovedMaintenanceLengthensTheCampaign(t *testing.T) {
	h, exec := execHarness(t)
	engineer := h.as(auth.RoleMaintenanceUser)
	planner := h.as(auth.RoleProductionPlanner)

	// A window somebody is still thinking about changes nothing.
	proposed, err := exec.SaveMaintenance(engineer, domain.MaintenanceWindow{
		FactoryID: h.seeded.FactoryID, StartDate: "2027-01-10", EndDate: "2027-01-12",
		Description: "Mill roller change",
	})
	if err != nil {
		t.Fatalf("save window: %v", err)
	}
	if proposed.Status != domain.MaintenancePlanned {
		t.Fatalf("a new window is PLANNED, got %s", proposed.Status)
	}

	before, err := h.planning.Generate(planner, h.seeded.BudgetID, service.GenerateRequest{Replace: true})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if before.LastDate != "2027-04-16" {
		t.Fatalf("baseline campaign ends %s, want 2027-04-16", before.LastDate)
	}
	if len(before.NonWorkingDays) != 0 {
		t.Errorf("a proposal must not remove crushing days, got %v", before.NonWorkingDays)
	}

	// An engineer cannot approve their own outage: approving one moves the end
	// of the season.
	proposed.Status = domain.MaintenanceApproved
	if _, err := exec.SaveMaintenance(engineer, proposed); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("approval needs the approver's permission, got %v", err)
	}

	approver := h.as(auth.RoleApprover, auth.RoleMaintenanceUser)
	approved, err := exec.SaveMaintenance(approver, proposed)
	if err != nil {
		t.Fatalf("approve window: %v", err)
	}
	if approved.ApprovedBy == "" {
		t.Error("an approved window must record who approved it")
	}

	after, err := h.planning.Generate(planner, h.seeded.BudgetID, service.GenerateRequest{Replace: true})
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if len(after.NonWorkingDays) != 3 {
		t.Errorf("non-working days = %v, want the three days of the outage", after.NonWorkingDays)
	}
	// The season is extended rather than shortened: the same cane still has to
	// be crushed, so the campaign ends three days later.
	if after.LastDate != "2027-04-19" {
		t.Errorf("campaign now ends %s, want 2027-04-19", after.LastDate)
	}
	if after.Summary.WorkingDays != before.Summary.WorkingDays {
		t.Errorf("working days changed from %d to %d; the outage must not cost tonnage",
			before.Summary.WorkingDays, after.Summary.WorkingDays)
	}
	if !after.Summary.CaneAllocated.Equal(before.Summary.CaneAllocated) {
		t.Errorf("cane allocated changed from %s to %s",
			before.Summary.CaneAllocated, after.Summary.CaneAllocated)
	}
}

func TestALineOutageDoesNotStopTheFactory(t *testing.T) {
	h, exec := execHarness(t)
	approver := h.as(auth.RoleApprover, auth.RoleMaintenanceUser)
	planner := h.as(auth.RoleProductionPlanner)

	lines, err := h.store.MasterData().Lines().List(approver, store.ListOptions{Top: 1})
	if err != nil || len(lines.Items) == 0 {
		t.Fatalf("no production line to test with: %v", err)
	}

	if _, err := exec.SaveMaintenance(approver, domain.MaintenanceWindow{
		FactoryID: h.seeded.FactoryID, LineID: lines.Items[0].ID,
		StartDate: "2027-01-10", EndDate: "2027-01-12",
		Description: "Packing line service", Status: domain.MaintenanceApproved,
	}); err != nil {
		t.Fatalf("save line window: %v", err)
	}

	result, err := h.planning.Generate(planner, h.seeded.BudgetID, service.GenerateRequest{Replace: true})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(result.NonWorkingDays) != 0 {
		t.Errorf("a line outage must not remove a crushing day, got %v", result.NonWorkingDays)
	}
	if result.LastDate != "2027-04-16" {
		t.Errorf("campaign ends %s, want the unchanged 2027-04-16", result.LastDate)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func stockFor(t *testing.T, ctx context.Context, exec *service.Execution,
	warehouseID, productID string,
) service.StockLine {
	t.Helper()
	lines, err := exec.Stock(ctx, store.ExecutionFilter{
		WarehouseID: warehouseID, ProductIDs: []string{productID},
	})
	if err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if len(lines) == 0 {
		return service.StockLine{
			WarehouseID: warehouseID, ProductID: productID,
			Quantity: domain.Zero, HoldQuantity: domain.Zero, Available: domain.Zero,
		}
	}
	return lines[0]
}

func seedOrder(t *testing.T, h *harness, exec *service.Execution,
	ctx context.Context, planned domain.Dec,
) domain.ProductionOrder {
	t.Helper()
	order, err := exec.CreateOrder(ctx, service.OrderRequest{
		FactoryID: h.seeded.FactoryID, BusinessDate: "2026-12-05",
		ProductID: h.seeded.Products["REF"], PlannedQty: planned,
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	return order
}

// releaseSeededPlan walks the budget version through its workflow to RELEASED
// and returns its id.
func releaseSeededPlan(t *testing.T, h *harness) string {
	t.Helper()
	planner := h.as(auth.RoleProductionPlanner)
	approver := h.as(auth.RoleApprover)

	detail, err := h.planning.GetVersion(planner, h.seeded.BudgetID)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	version := detail.Version
	steps := []struct {
		ctx    context.Context
		action domain.PlanAction
	}{
		{planner, domain.ActionSubmit},
		{approver, domain.ActionApprove},
		{approver, domain.ActionRelease},
	}
	for _, step := range steps {
		version, err = h.planning.Transition(step.ctx, h.seeded.BudgetID, service.TransitionRequest{
			Action: step.action, Reason: "test", RowVersion: version.RowVersion,
		})
		if err != nil {
			t.Fatalf("%s: %v", step.action, err)
		}
	}
	if version.Status != domain.StatusReleased {
		t.Fatalf("version is %s, want RELEASED", version.Status)
	}
	return version.ID
}
