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

	// The actuals container is released from the day it is created so that
	// operators can post to it. It is not a plan, and raising orders from it
	// would quietly produce nothing at all.
	_, err = exec.CreateOrdersFromPlan(ctx, h.seeded.ActualID, service.OrdersFromPlanRequest{
		From: "2026-12-01", To: "2026-12-03",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("the actuals container may not become orders, got %v", err)
	}
	if !strings.Contains(err.Error(), "actuals container") {
		t.Errorf("the refusal must say why: %v", err)
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

	// A parameter of the test's own, so the assertions are about the rules
	// rather than about whatever limits the demonstration scenario ships.
	pol, err := exec.SaveParameter(admin, domain.QualityParameter{
		Code: "TEST-PURITY", Name: "Purity", UOM: "PCT", Validity: domain.Validity{Active: true},
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
		Code: "TEST-TURBIDITY", Name: "Turbidity", UOM: "IU", Validity: domain.Validity{Active: true},
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
	if len(outcome.Unspecified) != 1 || outcome.Unspecified[0] != "TEST-TURBIDITY" {
		t.Errorf("the configuration gap must be reported, got %+v", outcome.Unspecified)
	}
}

func TestASampleTakesItsResultsOnlyOnce(t *testing.T) {
	h, exec := execHarness(t)
	admin := h.as(auth.RoleMasterDataAdmin)
	lab := h.as(auth.RoleQualityUser)

	pol, err := exec.SaveParameter(admin, domain.QualityParameter{
		Code: "TEST-POL", Name: "Polarisation", Validity: domain.Validity{Active: true},
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
	// The last day of cane, not the end of the campaign: the refinery and the
	// shipping gate run until September either way, and it is the crushing that
	// a maintenance window moves.
	if before.LastCrushingDate != "2027-04-16" {
		t.Fatalf("baseline crushing ends %s, want 2027-04-16", before.LastCrushingDate)
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
	// The crushing is extended rather than shortened: the same cane still has to
	// go through the mill, so it finishes three days later. The campaign end
	// does not move - the refinery was always going to run into September, and
	// three days of mill outage in December does not change that.
	if after.LastCrushingDate != "2027-04-19" {
		t.Errorf("crushing now ends %s, want 2027-04-19", after.LastCrushingDate)
	}
	if after.LastDate != before.LastDate {
		t.Errorf("the campaign end moved from %s to %s; a mill outage does not "+
			"move the end of the remelt season", before.LastDate, after.LastDate)
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
	if result.LastCrushingDate != "2027-04-16" {
		t.Errorf("crushing ends %s, want the unchanged 2027-04-16", result.LastCrushingDate)
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

// TestTheLatestSpecificationInForceWins pins how two overlapping limits are
// resolved. Tightening a limit without ending the older specification leaves
// both effective on the same day; the later start date is the one somebody most
// recently decided on, and leaving the choice to the repository's row order
// would make a verdict depend on the storage engine.
func TestTheLatestSpecificationInForceWins(t *testing.T) {
	h, exec := execHarness(t)
	admin := h.as(auth.RoleMasterDataAdmin)
	lab := h.as(auth.RoleQualityUser)
	product := h.seeded.Products["REF"]

	parameter, err := exec.SaveParameter(admin, domain.QualityParameter{
		Code: "TEST-GRAIN", Name: "Grain size", UOM: "MM", Validity: domain.Validity{Active: true},
	})
	if err != nil {
		t.Fatalf("save parameter: %v", err)
	}

	loose, tight := domain.D("0.900"), domain.D("0.500")
	for _, spec := range []domain.QualitySpec{
		{ProductID: product, ParameterID: parameter.ID, UpperLimit: &loose, ValidFrom: "2026-10-01"},
		{ProductID: product, ParameterID: parameter.ID, UpperLimit: &tight, ValidFrom: "2026-11-01"},
	} {
		if _, err := exec.SaveSpec(admin, spec); err != nil {
			t.Fatalf("save spec from %s: %v", spec.ValidFrom, err)
		}
	}

	effective, err := exec.SpecsFor(admin, product, "2026-12-06")
	if err != nil {
		t.Fatalf("specs for the date: %v", err)
	}
	mine := 0
	for _, spec := range effective {
		if spec.ParameterID == parameter.ID {
			mine++
		}
	}
	if mine != 2 {
		t.Fatalf("both specifications for the parameter are still in force, got %d", mine)
	}

	sample, err := exec.CreateSample(lab, service.SampleRequest{
		ProductID: product, FactoryID: h.seeded.FactoryID, BusinessDate: "2026-12-06",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}
	// 0.7 mm passes the older limit of 0.9 and fails the newer one of 0.5.
	outcome, err := exec.RecordResults(lab, sample.ID, service.ResultsRequest{
		Results:  []service.ResultInput{{ParameterID: parameter.ID, Value: domain.D("0.700")}},
		Complete: true,
	})
	if err != nil {
		t.Fatalf("record results: %v", err)
	}
	if outcome.Verdict != domain.QualityFail {
		t.Errorf("verdict = %s, want FAIL against the November limit of 0.5", outcome.Verdict)
	}
	if len(outcome.Sample.Results) != 1 || outcome.Sample.Results[0].UpperLimit == nil ||
		!outcome.Sample.Results[0].UpperLimit.Equal(tight) {
		t.Errorf("the result must record the limit it was judged against: %+v", outcome.Sample.Results)
	}
}

// TestTheSeededScenarioCanJudgeASample proves the demonstration data is usable
// for quality out of the box: without a parameter catalogue and limits in
// force, a laboratory sheet has nothing to judge against and every sample
// passes silently.
func TestTheSeededScenarioCanJudgeASample(t *testing.T) {
	h, exec := execHarness(t)
	admin := h.as(auth.RoleMasterDataAdmin)
	lab := h.as(auth.RoleQualityUser)

	parameters, err := exec.ListParameters(admin)
	if err != nil {
		t.Fatalf("list parameters: %v", err)
	}
	byCode := map[string]domain.QualityParameter{}
	for _, p := range parameters {
		byCode[p.Code] = p
	}
	for _, code := range []string{"POL", "COLOUR", "MOIST"} {
		if _, ok := byCode[code]; !ok {
			t.Fatalf("the seed must ship a %s parameter, got %v", code, byCode)
		}
	}

	specs, err := exec.SpecsFor(admin, h.seeded.Products["REF"], "2026-12-06")
	if err != nil {
		t.Fatalf("specs for refined sugar: %v", err)
	}
	if len(specs) < 3 {
		t.Fatalf("refined sugar has %d specifications in force, want at least 3", len(specs))
	}

	sample, err := exec.CreateSample(lab, service.SampleRequest{
		ProductID: h.seeded.Products["REF"], FactoryID: h.seeded.FactoryID,
		BusinessDate: "2026-12-06",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}
	// Colour of 60 IU is well outside the 45 IU limit for refined sugar.
	outcome, err := exec.RecordResults(lab, sample.ID, service.ResultsRequest{
		Results: []service.ResultInput{
			{ParameterID: byCode["POL"].ID, Value: domain.D("99.850")},
			{ParameterID: byCode["COLOUR"].ID, Value: domain.D("60")},
		},
		Complete: true,
	})
	if err != nil {
		t.Fatalf("record results: %v", err)
	}
	if outcome.Verdict != domain.QualityFail {
		t.Errorf("verdict = %s, want FAIL on colour", outcome.Verdict)
	}
	if len(outcome.Unspecified) != 0 {
		t.Errorf("both parameters are specified in the seed, got %v", outcome.Unspecified)
	}
}

// ---------------------------------------------------------------------------
// Certificate of analysis
// ---------------------------------------------------------------------------

func TestACertificateCarriesTheLimitsTheMaterialWasJudgedAgainst(t *testing.T) {
	h, exec := execHarness(t)
	lab := h.as(auth.RoleQualityUser)

	// A batch comes into existence when production makes it, so the batch is
	// confirmed first and the sample then names it by the code on the pallet
	// card - which is what a laboratory technician actually has.
	order := releasedOrder(t, h, exec, "", domain.D("330"))
	if _, err := exec.Confirm(h.as(auth.RoleProductionOperator, auth.RoleShiftSupervisor),
		order.ID, service.ConfirmRequest{
			BusinessDate: "2026-12-05", YieldQty: domain.D("330"), BatchCode: "B-2026-12-05-A",
		}); err != nil {
		t.Fatalf("confirm production of the batch: %v", err)
	}

	sample, err := exec.CreateSample(lab, service.SampleRequest{
		ProductID: h.seeded.Products["REF"], FactoryID: h.seeded.FactoryID,
		BusinessDate: "2026-12-05", BatchCode: "B-2026-12-05-A",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}
	parameters, err := exec.ListParameters(lab)
	if err != nil {
		t.Fatalf("list parameters: %v", err)
	}
	pol := parameterByCode(t, parameters, "POL")

	// A pol of 99.9 passes the refined specification comfortably.
	if _, err := exec.RecordResults(lab, sample.ID, service.ResultsRequest{
		Results:  []service.ResultInput{{ParameterID: pol, Value: domain.D("99.9")}},
		Complete: true,
	}); err != nil {
		t.Fatalf("record results: %v", err)
	}

	cert, err := exec.Certificate(h.as(auth.RoleExecutiveViewer), sample.ID)
	if err != nil {
		t.Fatalf("certificate: %v", err)
	}
	if cert.Verdict != domain.QualityPass {
		t.Errorf("verdict = %s, want PASS", cert.Verdict)
	}
	if len(cert.Lines) != 1 {
		t.Fatalf("one measurement, one line, got %d", len(cert.Lines))
	}
	line := cert.Lines[0]
	if line.ParameterCode != "POL" || line.ParameterName == "" {
		t.Errorf("the line must name the parameter: %+v", line)
	}
	if line.LowerLimit == nil {
		t.Error("a certificate without the limit is a number with nothing to judge it by")
	}
	if cert.ProductCode != "REF" || cert.CompanyName == "" || cert.FactoryName == "" {
		t.Errorf("the certificate must say whose material it is: %+v", cert)
	}
	// The certificate carries the batch, which is what ties it to a consignment.
	batch, err := h.store.Execution().BatchByCode(context.Background(), "B-2026-12-05-A")
	if err != nil {
		t.Fatalf("the confirmation must have created the batch: %v", err)
	}
	if cert.Sample.BatchID != batch.ID {
		t.Errorf("the certificate must name the batch, got %q want %q",
			cert.Sample.BatchID, batch.ID)
	}

	// A batch nobody produced cannot be sampled: that would be a sample of
	// nothing, and a certificate about nothing.
	if _, err := exec.CreateSample(lab, service.SampleRequest{
		ProductID: h.seeded.Products["REF"], FactoryID: h.seeded.FactoryID,
		BusinessDate: "2026-12-05", BatchCode: "B-NEVER-MADE",
	}); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("an unknown batch must be refused, got %v", err)
	}

	// Tightening the specification afterwards must not change what the customer
	// was already told.
	specs, err := exec.SpecsFor(lab, h.seeded.Products["REF"], "2026-12-05")
	if err != nil {
		t.Fatalf("specs: %v", err)
	}
	var polSpec domain.QualitySpec
	for _, s := range specs {
		if s.ParameterID == pol {
			polSpec = s
		}
	}
	tighter, warn := domain.D("99.95"), domain.D("99.97")
	polSpec.ID, polSpec.LowerLimit, polSpec.WarnLower = "", &tighter, &warn
	polSpec.ValidFrom = "2026-12-01"
	if _, err := exec.SaveSpec(h.as(auth.RoleQualityUser, auth.RoleMasterDataAdmin), polSpec); err != nil {
		t.Fatalf("tighten the specification: %v", err)
	}

	after, err := exec.Certificate(h.as(auth.RoleExecutiveViewer), sample.ID)
	if err != nil {
		t.Fatalf("certificate again: %v", err)
	}
	if after.Lines[0].LowerLimit.Equal(tighter) {
		t.Error("a certificate must keep the limit the material was judged against, " +
			"not the one in force today")
	}
	if after.Verdict != domain.QualityPass {
		t.Errorf("the verdict must not change either, got %s", after.Verdict)
	}
}

func TestAnUnfinishedSampleCannotBeCertified(t *testing.T) {
	h, exec := execHarness(t)
	lab := h.as(auth.RoleQualityUser)

	sample, err := exec.CreateSample(lab, service.SampleRequest{
		ProductID: h.seeded.Products["REF"], FactoryID: h.seeded.FactoryID,
		BusinessDate: "2026-12-05",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}
	if _, err := exec.Certificate(lab, sample.ID); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("an open sample must not produce a certificate, got %v", err)
	}

	parameters, err := exec.ListParameters(lab)
	if err != nil {
		t.Fatalf("list parameters: %v", err)
	}
	// An interim sheet is still not a statement the laboratory has made.
	if _, err := exec.RecordResults(lab, sample.ID, service.ResultsRequest{
		Results: []service.ResultInput{{
			ParameterID: parameterByCode(t, parameters, "POL"), Value: domain.D("99.9"),
		}},
	}); err != nil {
		t.Fatalf("record interim results: %v", err)
	}
	if _, err := exec.Certificate(lab, sample.ID); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("an unfinished sheet must not produce a certificate, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Bill-of-materials driven consumption
// ---------------------------------------------------------------------------

func TestAConfirmationWithNoComponentsTakesThemFromTheBOM(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.as(auth.RoleProductionOperator, auth.RoleShiftSupervisor)

	packaging, err := h.store.MasterData().PackagingTypes().GetByCode(context.Background(), "PJUMBO")
	if err != nil {
		t.Fatalf("packaging: %v", err)
	}
	order := releasedOrder(t, h, exec, packaging.ID, domain.D("330"))

	// 330 t of jumbo bags at 1.10 t each is 300 bags, plus the bag's 1 % scrap
	// allowance: 303.
	result, err := exec.Confirm(ctx, order.ID, service.ConfirmRequest{
		BusinessDate: order.BusinessDate, YieldQty: domain.D("330"),
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}

	byCode := map[string]domain.Dec{}
	for _, c := range result.Confirmation.Consumptions {
		mat, err := h.store.MasterData().Materials().Get(context.Background(), c.MaterialID)
		if err != nil {
			t.Fatalf("material: %v", err)
		}
		byCode[mat.Code] = c.Quantity
	}

	if got, ok := byCode["BAG-JUMBO"]; !ok || !got.Equal(domain.D("303")) {
		t.Errorf("jumbo bags = %s, want 303 (300 bags plus 1 %% scrap)", got)
	}
	// One liner per bag, and the liner has its own 1 % scrap rate: 303 * 1.01.
	if got, ok := byCode["LINER"]; !ok || !got.Equal(domain.D("306.03")) {
		t.Errorf("liners = %s, want 306.030", got)
	}
	// Thread at 0.004 spools per bag with a 3 % scrap rate: 303 * 0.004 * 1.03.
	if got, ok := byCode["THREAD"]; !ok || !got.Equal(domain.D("1.248")) {
		t.Errorf("thread = %s, want 1.248 spools", got)
	}
	if len(byCode) != 3 {
		t.Errorf("the jumbo bag consumes a bag, a liner and thread, got %v", byCode)
	}
}

func TestComponentsEnteredByHandAreNotOverriddenByTheBOM(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.as(auth.RoleProductionOperator, auth.RoleShiftSupervisor)

	packaging, err := h.store.MasterData().PackagingTypes().GetByCode(context.Background(), "PJUMBO")
	if err != nil {
		t.Fatalf("packaging: %v", err)
	}
	liner, err := h.store.MasterData().Materials().GetByCode(context.Background(), "LINER")
	if err != nil {
		t.Fatalf("material: %v", err)
	}
	order := releasedOrder(t, h, exec, packaging.ID, domain.D("330"))

	// An operator who counted what actually went out of the store is a better
	// source than a bill of materials, so what they entered stands alone.
	result, err := exec.Confirm(ctx, order.ID, service.ConfirmRequest{
		BusinessDate: order.BusinessDate, YieldQty: domain.D("330"),
		Consumptions: []service.ConsumptionInput{{
			MaterialID: liner.ID, Quantity: domain.D("290"), UOM: "EA",
		}},
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if len(result.Confirmation.Consumptions) != 1 {
		t.Fatalf("the entered line must stand alone, got %v", result.Confirmation.Consumptions)
	}
	if !result.Confirmation.Consumptions[0].Quantity.Equal(domain.D("290")) {
		t.Errorf("the counted quantity must survive, got %s",
			result.Confirmation.Consumptions[0].Quantity)
	}
}

func TestBulkProductionConsumesNoPackaging(t *testing.T) {
	h, exec := execHarness(t)
	ctx := h.as(auth.RoleProductionOperator, auth.RoleShiftSupervisor)

	// Raw sugar going straight to the refinery is not bagged at all.
	order := releasedOrder(t, h, exec, "", domain.D("500"))
	result, err := exec.Confirm(ctx, order.ID, service.ConfirmRequest{
		BusinessDate: order.BusinessDate, YieldQty: domain.D("500"),
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if len(result.Confirmation.Consumptions) != 0 {
		t.Errorf("an order with no packaging must not invent a bag, got %v",
			result.Confirmation.Consumptions)
	}
}

// releasedOrder creates an order and releases it, which is the state a
// confirmation needs.
func releasedOrder(t *testing.T, h *harness, exec *service.Execution,
	packagingID string, qty domain.Dec,
) domain.ProductionOrder {

	t.Helper()
	ctx := h.as(auth.RoleProductionOperator, auth.RoleShiftSupervisor)

	order, err := exec.CreateOrder(ctx, service.OrderRequest{
		FactoryID: h.seeded.FactoryID, BusinessDate: "2026-12-05",
		ProductID: h.seeded.Products["REF"], PackagingID: packagingID, PlannedQty: qty,
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	order, err = exec.ActOnOrder(ctx, order.ID, service.OrderActionRequest{
		Action: domain.OrderActionRelease,
	})
	if err != nil {
		t.Fatalf("release order: %v", err)
	}
	return order
}
