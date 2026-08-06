package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// This file is the execution half of the conformance suite: production orders,
// confirmations, inventory postings, quality and maintenance. The rules it
// pins down are the ones the service layer relies on and that the two
// implementations could plausibly disagree about - business keys, optimistic
// concurrency, upsert semantics, and what a rolled-back posting leaves behind.

// execFixture is the master data the execution tests need on top of the
// planning fixture.
type execFixture struct {
	fixture
	line, shift, warehouse2, material, reason string
}

func seedExecFixture(t *testing.T, ctx context.Context, s store.Store) execFixture {
	t.Helper()
	base := seedFixture(t, ctx, s)
	md := s.MasterData()

	line, err := md.Lines().Save(ctx, domain.ProductionLine{
		FactoryID: base.factory, Code: "REF-1", Name: "Refinery line 1",
		Stage: domain.StageRefining, RatedTPH: domain.D("25"),
		Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save line")

	shift, err := md.Shifts().Save(ctx, domain.Shift{
		FactoryID: base.factory, Code: "A", Name: "Morning", StartTime: "06:00",
		Hours: domain.D("8"), Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save shift")

	warehouse2, err := md.Warehouses().Save(ctx, domain.Warehouse{
		FactoryID: base.factory, Code: "FG-WH3", Name: "Finished Goods Warehouse 3",
		StorageClass: domain.StorageFinished, CapacityTons: domain.D("47000"),
		UsablePct: domain.D("100"), Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save second warehouse")

	material, err := md.Materials().Save(ctx, domain.Material{
		Code: "PP-BAG-50", Name: "Polypropylene bag 50 kg", UOM: "TON",
		Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save material")

	reason, err := md.ReasonCodes().Save(ctx, domain.ReasonCode{
		Code: "VAR-YIELD", Name: "Yield variance", Category: "VARIANCE",
		Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save reason code")

	return execFixture{base, line.ID, shift.ID, warehouse2.ID, material.ID, reason.Code}
}

// ---------------------------------------------------------------------------
// Production orders
// ---------------------------------------------------------------------------

func testOrders(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedExecFixture(t, ctx, s)
	exec := s.Execution()

	first, err := exec.NextNumber(ctx, store.SeriesOrder, "F1", 2026)
	must(t, err, "next order number")
	if first != "PO-F1-2026-00001" {
		t.Fatalf("first order number = %q, want PO-F1-2026-00001", first)
	}
	if _, err := exec.NextNumber(ctx, "XX", "F1", 2026); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("an unknown series must be refused, got %v", err)
	}

	order, err := exec.SaveOrder(ctx, domain.ProductionOrder{
		OrderNo: first, CompanyID: f.company, FactoryID: f.factory, LineID: f.line,
		BusinessDate: "2026-12-01", ShiftID: f.shift, ProductID: f.product,
		PlannedQty: domain.D("300"), Status: domain.OrderPlanned, Team: "A",
	}, "supervisor")
	must(t, err, "save order")
	if order.ID == "" || order.RowVersion != 1 {
		t.Fatalf("saved order = %+v", order)
	}
	if order.Priority != 5 {
		t.Errorf("priority defaults to 5, got %d", order.Priority)
	}
	if order.CreatedBy != "supervisor" || order.CreatedAt.IsZero() {
		t.Error("audit fields must be populated on insert")
	}

	// The numbering is derived from what exists, so the next call moves on.
	second, err := exec.NextNumber(ctx, store.SeriesOrder, "F1", 2026)
	must(t, err, "next order number again")
	if second != "PO-F1-2026-00002" {
		t.Errorf("second order number = %q, want PO-F1-2026-00002", second)
	}
	// Each series counts on its own, and each year starts again at one.
	for _, c := range []struct{ series, want string }{
		{store.SeriesConfirmation, "CF-F1-2026-00001"},
		{store.SeriesDocument, "MD-F1-2026-00001"},
		{store.SeriesSample, "QS-F1-2026-00001"},
	} {
		got, err := exec.NextNumber(ctx, c.series, "F1", 2026)
		must(t, err, "next "+c.series+" number")
		if got != c.want {
			t.Errorf("next %s number = %q, want %q", c.series, got, c.want)
		}
	}
	nextYear, err := exec.NextNumber(ctx, store.SeriesOrder, "F1", 2027)
	must(t, err, "next order number for the following year")
	if nextYear != "PO-F1-2027-00001" {
		t.Errorf("the numbering restarts each year, got %q", nextYear)
	}

	if _, err := exec.SaveOrder(ctx, domain.ProductionOrder{
		OrderNo: first, CompanyID: f.company, FactoryID: f.factory,
		BusinessDate: "2026-12-02", ProductID: f.product, PlannedQty: domain.D("10"),
		Status: domain.OrderPlanned,
	}, "supervisor"); !errors.Is(err, domain.ErrDuplicate) {
		t.Errorf("a repeated order number must be a duplicate, got %v", err)
	}

	got, err := exec.GetOrder(ctx, order.ID)
	must(t, err, "get order")
	if !got.PlannedQty.Equal(domain.D("300")) || got.LineID != f.line || got.ShiftID != f.shift {
		t.Errorf("round trip lost data: %+v", got)
	}
	if got.Status != domain.OrderPlanned || got.Team != "A" {
		t.Errorf("round trip lost the status or the team: %+v", got)
	}

	// Release, then confirm part of it.
	got.Status = domain.OrderReleased
	released, err := exec.SaveOrder(ctx, got, "supervisor")
	must(t, err, "release order")
	if released.RowVersion != 2 || released.UpdatedBy != "supervisor" {
		t.Errorf("released order = %+v", released)
	}
	if !released.CreatedAt.Equal(order.CreatedAt) || released.CreatedBy != "supervisor" {
		t.Error("an update must preserve the creation stamp")
	}

	// The stale copy the client still holds must be refused.
	if _, err := exec.SaveOrder(ctx, got, "supervisor"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("a stale row version must conflict, got %v", err)
	}

	released.ConfirmedQty = domain.D("240")
	released.Status = domain.OrderPartiallyConfirmed
	released.VarianceReason = f.reason
	confirmed, err := exec.SaveOrder(ctx, released, "supervisor")
	must(t, err, "confirm order")
	if confirmed.VarianceReason != f.reason {
		t.Errorf("variance reason = %q, want %q", confirmed.VarianceReason, f.reason)
	}
	if !confirmed.OpenQty().Equal(domain.D("60")) {
		t.Errorf("open quantity = %s, want 60", confirmed.OpenQty())
	}

	open, err := exec.ListOrders(ctx, store.ExecutionFilter{
		FactoryID: f.factory, From: "2026-12-01", To: "2026-12-31", OpenOnly: true,
	})
	must(t, err, "list open orders")
	if open.Count != 1 || len(open.Items) != 1 || open.Items[0].OrderNo != first {
		t.Errorf("open orders = %+v (count %d)", open.Items, open.Count)
	}

	byStatus, err := exec.ListOrders(ctx, store.ExecutionFilter{
		FactoryID: f.factory, Statuses: []string{string(domain.OrderCompleted)},
	})
	must(t, err, "list completed orders")
	if byStatus.Count != 0 {
		t.Errorf("no order is completed yet, got %d", byStatus.Count)
	}

	outOfRange, err := exec.ListOrders(ctx, store.ExecutionFilter{
		FactoryID: f.factory, From: "2027-01-01", To: "2027-01-31",
	})
	must(t, err, "list orders out of range")
	if outOfRange.Count != 0 {
		t.Errorf("the date filter must exclude the order, got %d", outOfRange.Count)
	}

	if _, err := exec.GetOrder(ctx, "3f1d1a1e-0000-4000-8000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("an unknown order must be not found, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Confirmations
// ---------------------------------------------------------------------------

func testConfirmations(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedExecFixture(t, ctx, s)
	exec := s.Execution()

	order, err := exec.SaveOrder(ctx, domain.ProductionOrder{
		OrderNo: "PO-F1-2026-00001", CompanyID: f.company, FactoryID: f.factory,
		BusinessDate: "2026-12-01", ProductID: f.product, PlannedQty: domain.D("300"),
		Status: domain.OrderReleased,
	}, "supervisor")
	must(t, err, "save order")

	saved, err := exec.SaveConfirmation(ctx, domain.ProductionConfirmation{
		OrderID: order.ID, ConfirmationNo: "CF-2026-00001", BusinessDate: "2026-12-01",
		ShiftID: f.shift, YieldQty: domain.D("240"), ScrapQty: domain.D("2.5"),
		LabourHours: domain.D("24"), MachineHours: domain.D("8"),
		Consumptions: []domain.MaterialConsumption{
			{MaterialID: f.material, Quantity: domain.D("4.8"), UOM: "TON"},
		},
	}, "operator")
	must(t, err, "save confirmation")
	if saved.ID == "" || saved.RowVersion != 1 {
		t.Fatalf("saved confirmation = %+v", saved)
	}
	if len(saved.Consumptions) != 1 || saved.Consumptions[0].ConfirmationID != saved.ID {
		t.Errorf("the consumption must be linked to the confirmation: %+v", saved.Consumptions)
	}

	got, err := exec.GetConfirmation(ctx, saved.ID)
	must(t, err, "get confirmation")
	if !got.YieldQty.Equal(domain.D("240")) || !got.ScrapQty.Equal(domain.D("2.5")) {
		t.Errorf("round trip lost the quantities: %+v", got)
	}
	if !got.LabourHours.Equal(domain.D("24")) || !got.MachineHours.Equal(domain.D("8")) {
		t.Errorf("round trip lost the hours: %+v", got)
	}
	if len(got.Consumptions) != 1 || !got.Consumptions[0].Quantity.Equal(domain.D("4.8")) {
		t.Errorf("round trip lost the consumption: %+v", got.Consumptions)
	}

	if _, err := exec.SaveConfirmation(ctx, domain.ProductionConfirmation{
		OrderID: order.ID, ConfirmationNo: "CF-2026-00001", BusinessDate: "2026-12-02",
		YieldQty: domain.D("10"),
	}, "operator"); !errors.Is(err, domain.ErrDuplicate) {
		t.Errorf("a repeated confirmation number must be a duplicate, got %v", err)
	}

	list, err := exec.ListConfirmations(ctx, order.ID)
	must(t, err, "list confirmations")
	if len(list) != 1 || list[0].ConfirmationNo != "CF-2026-00001" {
		t.Fatalf("confirmations = %+v", list)
	}

	// A confirmation is never edited; the one change it takes is being flagged
	// as reversed once its counter-confirmation is posted.
	got.Reversed = true
	if _, err := exec.SaveConfirmation(ctx, got, "supervisor"); err != nil {
		t.Fatalf("flag confirmation reversed: %v", err)
	}
	after, err := exec.GetConfirmation(ctx, saved.ID)
	must(t, err, "get reversed confirmation")
	if !after.Reversed {
		t.Error("the confirmation must be flagged as reversed")
	}

	if _, err := exec.GetConfirmation(ctx, "3f1d1a1e-0000-4000-8000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("an unknown confirmation must be not found, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Inventory
// ---------------------------------------------------------------------------

// post is a small helper so that the inventory test reads as a sequence of
// warehouse movements rather than as struct literals.
func post(t *testing.T, ctx context.Context, exec store.Execution, no string,
	docType domain.DocType, f execFixture, items ...domain.InventoryDocumentItem,
) domain.InventoryDocument {
	t.Helper()
	for i := range items {
		items[i].LineNo = i + 1
		items[i].ProductID = f.product
		items[i].UOM = "TON"
	}
	d, err := exec.PostDocument(ctx, domain.InventoryDocument{
		DocumentNo: no, DocType: docType, BusinessDate: "2026-12-01",
		FactoryID: f.factory, Items: items,
	}, "keeper")
	must(t, err, "post "+no)
	return d
}

func balance(t *testing.T, ctx context.Context, exec store.Execution,
	warehouse, product string,
) domain.StockPosition {
	t.Helper()
	positions, err := exec.Positions(ctx, [][2]string{{warehouse, product}})
	must(t, err, "read position")
	return positions[warehouse+"|"+product]
}

func testInventory(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedExecFixture(t, ctx, s)
	exec := s.Execution()

	// A pair that has never been posted reads as zero, not as missing: the
	// first receipt into a new store must not be a special case.
	empty := balance(t, ctx, exec, f.warehouse, f.product)
	if !empty.Quantity.IsZero() || empty.WarehouseID != f.warehouse {
		t.Fatalf("an unposted pair must read as a zero position, got %+v", empty)
	}

	receipt := post(t, ctx, exec, "MD-2026-00001", domain.DocReceipt, f,
		domain.InventoryDocumentItem{WarehouseID: f.warehouse, Quantity: domain.D("500")})
	if receipt.ID == "" || receipt.PostedAt.IsZero() || receipt.CreatedBy != "keeper" {
		t.Fatalf("posted document = %+v", receipt)
	}
	if len(receipt.Items) != 1 || receipt.Items[0].DocumentID != receipt.ID {
		t.Fatalf("the item must be linked to the document: %+v", receipt.Items)
	}
	if after := balance(t, ctx, exec, f.warehouse, f.product); !after.Quantity.Equal(domain.D("500")) {
		t.Errorf("balance after the receipt = %s, want 500", after.Quantity)
	}

	post(t, ctx, exec, "MD-2026-00002", domain.DocIssue, f,
		domain.InventoryDocumentItem{WarehouseID: f.warehouse, Quantity: domain.D("-120")})
	if after := balance(t, ctx, exec, f.warehouse, f.product); !after.Quantity.Equal(domain.D("380")) {
		t.Errorf("balance after the issue = %s, want 380", after.Quantity)
	}

	// A hold moves the held quantity, not the balance: the sugar is still in
	// the shed, it just may not leave it.
	post(t, ctx, exec, "MD-2026-00003", domain.DocHold, f,
		domain.InventoryDocumentItem{WarehouseID: f.warehouse, Quantity: domain.D("100")})
	held := balance(t, ctx, exec, f.warehouse, f.product)
	if !held.Quantity.Equal(domain.D("380")) || !held.HoldQuantity.Equal(domain.D("100")) {
		t.Errorf("after the hold: quantity %s hold %s, want 380 / 100", held.Quantity, held.HoldQuantity)
	}
	if !held.Available().Equal(domain.D("280")) {
		t.Errorf("available = %s, want 280", held.Available())
	}

	post(t, ctx, exec, "MD-2026-00004", domain.DocRelease, f,
		domain.InventoryDocumentItem{WarehouseID: f.warehouse, Quantity: domain.D("100")})
	if after := balance(t, ctx, exec, f.warehouse, f.product); !after.HoldQuantity.IsZero() {
		t.Errorf("hold after the release = %s, want 0", after.HoldQuantity)
	}

	// A transfer is one document with two signed lines, so both balances move
	// together or neither does.
	post(t, ctx, exec, "MD-2026-00005", domain.DocTransfer, f,
		domain.InventoryDocumentItem{WarehouseID: f.warehouse, Quantity: domain.D("-50")},
		domain.InventoryDocumentItem{WarehouseID: f.warehouse2, Quantity: domain.D("50")})
	if from := balance(t, ctx, exec, f.warehouse, f.product); !from.Quantity.Equal(domain.D("330")) {
		t.Errorf("source balance = %s, want 330", from.Quantity)
	}
	if to := balance(t, ctx, exec, f.warehouse2, f.product); !to.Quantity.Equal(domain.D("50")) {
		t.Errorf("target balance = %s, want 50", to.Quantity)
	}

	if _, err := exec.PostDocument(ctx, domain.InventoryDocument{
		DocumentNo: "MD-2026-00001", DocType: domain.DocReceipt, BusinessDate: "2026-12-02",
		FactoryID: f.factory,
		Items: []domain.InventoryDocumentItem{{
			LineNo: 1, WarehouseID: f.warehouse, ProductID: f.product,
			Quantity: domain.D("1"), UOM: "TON",
		}},
	}, "keeper"); !errors.Is(err, domain.ErrDuplicate) {
		t.Errorf("a repeated document number must be a duplicate, got %v", err)
	}

	touching, err := exec.ListDocuments(ctx, store.ExecutionFilter{
		FactoryID: f.factory, WarehouseID: f.warehouse2,
	})
	must(t, err, "list documents by warehouse")
	if touching.Count != 1 || len(touching.Items) != 1 ||
		touching.Items[0].DocumentNo != "MD-2026-00005" {
		t.Errorf("only the transfer touches the second warehouse, got %+v", touching.Items)
	}
	if len(touching.Items[0].Items) != 2 {
		t.Errorf("a listed document must carry its lines, got %d", len(touching.Items[0].Items))
	}

	all, err := exec.ListDocuments(ctx, store.ExecutionFilter{
		FactoryID: f.factory, From: "2026-12-01", To: "2026-12-01",
	})
	must(t, err, "list documents by date")
	if all.Count != 5 {
		t.Errorf("documents on the day = %d, want 5", all.Count)
	}

	// --- reversal ---
	// Nothing is deleted and nothing is edited: the counter-document is posted,
	// and the original is flagged so that it cannot be reversed twice.
	//
	// The receipt is reversed after the stock it brought in has already moved
	// on, which drives the balance negative. The store must allow that: whether
	// a negative balance is acceptable is a business decision the domain takes
	// against the caller's permission, not something the ledger forbids
	// outright.
	counter, err := domain.ReverseDocument(receipt, "2026-12-02", "posted to the wrong store", "supervisor")
	must(t, err, "build reversal")
	counter.DocumentNo = "MD-2026-00006"
	if _, err := exec.PostDocument(ctx, counter, "supervisor"); err != nil {
		t.Fatalf("post reversal: %v", err)
	}
	if after := balance(t, ctx, exec, f.warehouse, f.product); !after.Quantity.Equal(domain.D("-170")) {
		t.Errorf("balance after the reversal = %s, want -170", after.Quantity)
	}

	if err := exec.MarkReversed(ctx, receipt.ID, "supervisor"); err != nil {
		t.Fatalf("mark reversed: %v", err)
	}
	reversed, err := exec.GetDocument(ctx, receipt.ID)
	must(t, err, "get reversed document")
	if !reversed.Reversed {
		t.Error("the original must be flagged as reversed")
	}
	if err := exec.MarkReversed(ctx, receipt.ID, "supervisor"); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("a second reversal must be refused, got %v", err)
	}
	if err := exec.MarkReversed(ctx, "3f1d1a1e-0000-4000-8000-000000000000", "supervisor"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("reversing an unknown document must be not found, got %v", err)
	}
	if _, err := domain.ReverseDocument(reversed, "2026-12-03", "again", "supervisor"); err == nil {
		t.Error("the domain must refuse to build a second reversal of the same document")
	}

	positions, err := exec.ListPositions(ctx, store.ExecutionFilter{WarehouseID: f.warehouse2})
	must(t, err, "list positions")
	if len(positions) != 1 || positions[0].WarehouseID != f.warehouse2 {
		t.Errorf("positions in the second warehouse = %+v", positions)
	}
}

// testInventoryRollback proves that a posting abandoned mid-transaction leaves
// neither a document nor a balance behind. It is the property the whole
// execution layer leans on, and the one most easily broken by a store that
// hands out shallow copies of its state.
func testInventoryRollback(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedExecFixture(t, ctx, s)

	post(t, ctx, s.Execution(), "MD-2026-00001", domain.DocReceipt, f,
		domain.InventoryDocumentItem{WarehouseID: f.warehouse, Quantity: domain.D("500")})

	boom := errors.New("the shift supervisor changed their mind")
	err := s.InTx(ctx, func(tx store.Store) error {
		if _, err := tx.Execution().PostDocument(ctx, domain.InventoryDocument{
			DocumentNo: "MD-2026-00002", DocType: domain.DocIssue, BusinessDate: "2026-12-01",
			FactoryID: f.factory,
			Items: []domain.InventoryDocumentItem{{
				LineNo: 1, WarehouseID: f.warehouse, ProductID: f.product,
				Quantity: domain.D("-200"), UOM: "TON",
			}},
		}, "keeper"); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("InTx must return the callback error, got %v", err)
	}

	if after := balance(t, ctx, s.Execution(), f.warehouse, f.product); !after.Quantity.Equal(domain.D("500")) {
		t.Errorf("balance after the rollback = %s, want the original 500", after.Quantity)
	}
	documents, err := s.Execution().ListDocuments(ctx, store.ExecutionFilter{FactoryID: f.factory})
	must(t, err, "list documents after the rollback")
	if documents.Count != 1 {
		t.Errorf("documents after the rollback = %d, want only the committed receipt", documents.Count)
	}
}

// ---------------------------------------------------------------------------
// Quality
// ---------------------------------------------------------------------------

func testQuality(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedExecFixture(t, ctx, s)
	exec := s.Execution()

	pol, err := exec.SaveParameter(ctx, domain.QualityParameter{
		Code: "POL", Name: "Polarisation", UOM: "PCT", TestMethod: "ICUMSA GS1-1",
		Validity: domain.Validity{Active: true},
	}, "lab")
	must(t, err, "save parameter")
	if pol.ID == "" || pol.RowVersion != 1 {
		t.Fatalf("saved parameter = %+v", pol)
	}

	// The code is the business key: saving it again updates the same row.
	again, err := exec.SaveParameter(ctx, domain.QualityParameter{
		Code: "POL", Name: "Polarisation (pol %)", UOM: "PCT",
		Validity: domain.Validity{Active: true},
	}, "lab")
	must(t, err, "save parameter again")
	if again.ID != pol.ID || again.RowVersion != 2 {
		t.Errorf("re-saving by code must update in place, got %+v", again)
	}
	parameters, err := exec.ListParameters(ctx)
	must(t, err, "list parameters")
	if len(parameters) != 1 || parameters[0].Name != "Polarisation (pol %)" {
		t.Errorf("parameters = %+v", parameters)
	}

	lower, upper := domain.D("99.500"), domain.D("100.000")
	warn := domain.D("99.700")
	spec, err := exec.SaveSpec(ctx, domain.QualitySpec{
		ProductID: f.product, ParameterID: pol.ID,
		LowerLimit: &lower, UpperLimit: &upper, WarnLower: &warn,
		ValidFrom: "2026-10-01",
	}, "quality")
	must(t, err, "save spec")
	if spec.ID == "" || spec.RowVersion != 1 {
		t.Fatalf("saved spec = %+v", spec)
	}

	effective, err := exec.SpecsFor(ctx, f.product, "2026-12-01")
	must(t, err, "specs on a date inside the validity")
	if len(effective) != 1 || effective[0].LowerLimit == nil ||
		!effective[0].LowerLimit.Equal(lower) {
		t.Fatalf("effective specs = %+v", effective)
	}

	// Effective dating is the point: last season is judged by last season's
	// limits, so a specification that starts in October does not apply in May.
	before, err := exec.SpecsFor(ctx, f.product, "2026-05-01")
	must(t, err, "specs before the validity")
	if len(before) != 0 {
		t.Errorf("a specification must not apply before it is effective, got %+v", before)
	}

	// Re-saving the same product, parameter and start date updates in place
	// rather than creating a second, conflicting specification.
	spec.ID = ""
	spec.UpperLimit = &upper
	updated, err := exec.SaveSpec(ctx, spec, "quality")
	must(t, err, "re-save spec")
	if updated.ID != effective[0].ID || updated.RowVersion != 2 {
		t.Errorf("re-saving a specification must update in place, got %+v", updated)
	}

	sample, err := exec.SaveSample(ctx, domain.QualitySample{
		SampleNo: "QS-2026-00001", ProductID: f.product, FactoryID: f.factory,
		BusinessDate: "2026-12-01", ShiftID: f.shift, LabUser: "lab",
	}, "lab")
	must(t, err, "save sample")
	if sample.Status != "OPEN" || sample.TakenAt.IsZero() {
		t.Errorf("a new sample defaults to OPEN and stamps the time, got %+v", sample)
	}

	must(t, exec.SaveResults(ctx, sample.ID, []domain.QualityResult{{
		ParameterID: pol.ID, Value: domain.D("99.600"), UOM: "PCT",
		LowerLimit: &lower, UpperLimit: &upper,
		Status: domain.EvaluateResult(domain.D("99.600"), updated),
	}}, "lab"), "save results")

	withResults, err := exec.GetSample(ctx, sample.ID)
	must(t, err, "get sample")
	if len(withResults.Results) != 1 {
		t.Fatalf("sample results = %+v", withResults.Results)
	}
	result := withResults.Results[0]
	if !result.Value.Equal(domain.D("99.600")) || result.SampleID != sample.ID {
		t.Errorf("round trip lost the measurement: %+v", result)
	}
	if result.LowerLimit == nil || !result.LowerLimit.Equal(lower) {
		t.Error("a result must keep the limits it was judged against")
	}
	if result.Status != domain.QualityWarning || withResults.Verdict() != domain.QualityWarning {
		t.Errorf("99.6 is inside the hard limits but below the warning band: %s / %s",
			result.Status, withResults.Verdict())
	}

	// Re-entering the sheet replaces the set: a parameter the laboratory has
	// dropped must not keep voting in the verdict.
	must(t, exec.SaveResults(ctx, sample.ID, nil, "lab"), "clear results")
	cleared, err := exec.GetSample(ctx, sample.ID)
	must(t, err, "get cleared sample")
	if len(cleared.Results) != 0 {
		t.Errorf("results after a replacement with none = %+v", cleared.Results)
	}

	if err := exec.SaveResults(ctx, "3f1d1a1e-0000-4000-8000-000000000000", nil, "lab"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("results for an unknown sample must be not found, got %v", err)
	}

	samples, err := exec.ListSamples(ctx, store.ExecutionFilter{
		FactoryID: f.factory, From: "2026-12-01", To: "2026-12-31",
		ProductIDs: []string{f.product},
	})
	must(t, err, "list samples")
	if samples.Count != 1 || len(samples.Items) != 1 {
		t.Errorf("samples = %+v (count %d)", samples.Items, samples.Count)
	}

	// --- holds ---
	hold, err := exec.SaveHold(ctx, domain.QualityHold{
		WarehouseID: f.warehouse, ProductID: f.product, SampleID: sample.ID,
		Quantity: domain.D("100"), PlacedOn: "2026-12-01", Reason: "colour out of specification",
	}, "quality")
	must(t, err, "save hold")
	if !hold.IsOpen() || hold.RowVersion != 1 {
		t.Fatalf("saved hold = %+v", hold)
	}

	open, err := exec.ListHolds(ctx, store.ExecutionFilter{
		WarehouseID: f.warehouse, ProductIDs: []string{f.product}, OpenOnly: true,
	})
	must(t, err, "list open holds")
	if len(open) != 1 {
		t.Fatalf("open holds = %+v", open)
	}

	stale := hold
	hold.ReleasedOn, hold.ReleasedBy = "2026-12-03", "quality"
	released, err := exec.SaveHold(ctx, hold, "quality")
	must(t, err, "release hold")
	if released.IsOpen() || released.RowVersion != 2 {
		t.Errorf("released hold = %+v", released)
	}
	if _, err := exec.SaveHold(ctx, stale, "quality"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("a stale hold must conflict, got %v", err)
	}

	stillOpen, err := exec.ListHolds(ctx, store.ExecutionFilter{OpenOnly: true})
	must(t, err, "list open holds after the release")
	if len(stillOpen) != 0 {
		t.Errorf("no hold is open any more, got %+v", stillOpen)
	}
	if _, err := exec.GetHold(ctx, "3f1d1a1e-0000-4000-8000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("an unknown hold must be not found, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Maintenance
// ---------------------------------------------------------------------------

func testMaintenance(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedExecFixture(t, ctx, s)
	exec := s.Execution()

	proposed, err := exec.SaveMaintenance(ctx, domain.MaintenanceWindow{
		FactoryID: f.factory, StartDate: "2027-01-10", EndDate: "2027-01-12",
		Description: "Mill roller change",
	}, "engineering")
	must(t, err, "save maintenance window")
	if proposed.Status != domain.MaintenancePlanned || proposed.RowVersion != 1 {
		t.Fatalf("a new window is PLANNED, got %+v", proposed)
	}

	approvedWindow, err := exec.SaveMaintenance(ctx, domain.MaintenanceWindow{
		FactoryID: f.factory, StartDate: "2027-02-01", EndDate: "2027-02-02",
		Description: "Boiler inspection", Status: domain.MaintenanceApproved,
		ApprovedBy: "plant-manager",
	}, "engineering")
	must(t, err, "save approved window")

	// A line-specific outage does not stop the factory, so it must not become a
	// non-working day even when it is approved.
	if _, err := exec.SaveMaintenance(ctx, domain.MaintenanceWindow{
		FactoryID: f.factory, LineID: f.line, StartDate: "2027-02-10", EndDate: "2027-02-10",
		Description: "Packing line service", Status: domain.MaintenanceApproved,
		ApprovedBy: "plant-manager",
	}, "engineering"); err != nil {
		t.Fatalf("save line window: %v", err)
	}

	all, err := exec.ListMaintenance(ctx, store.ExecutionFilter{FactoryID: f.factory})
	must(t, err, "list maintenance")
	if len(all) != 3 || all[0].StartDate != "2027-01-10" {
		t.Fatalf("windows = %+v", all)
	}

	approved, err := exec.ListMaintenance(ctx, store.ExecutionFilter{
		FactoryID: f.factory, Statuses: []string{domain.MaintenanceApproved},
	})
	must(t, err, "list approved maintenance")
	if len(approved) != 2 {
		t.Errorf("approved windows = %d, want 2", len(approved))
	}

	// The generator asks for the windows overlapping the season, so the filter
	// has to catch a window that straddles the boundary rather than one that
	// merely starts inside it.
	overlapping, err := exec.ListMaintenance(ctx, store.ExecutionFilter{
		FactoryID: f.factory, From: "2027-01-11", To: "2027-01-11",
	})
	must(t, err, "list overlapping maintenance")
	if len(overlapping) != 1 || overlapping[0].ID != proposed.ID {
		t.Errorf("the straddling window must be found, got %+v", overlapping)
	}

	outside, err := exec.ListMaintenance(ctx, store.ExecutionFilter{
		FactoryID: f.factory, From: "2027-03-01", To: "2027-03-31",
	})
	must(t, err, "list maintenance outside the range")
	if len(outside) != 0 {
		t.Errorf("no window touches March, got %+v", outside)
	}

	nonWorking := domain.NonWorkingDays(all)
	if len(nonWorking) != 2 {
		t.Fatalf("non-working days = %v, want the two days of the approved factory window", nonWorking)
	}
	for _, d := range []domain.BusinessDate{"2027-02-01", "2027-02-02"} {
		if !nonWorking[d] {
			t.Errorf("%s must be a non-working day", d)
		}
	}
	if nonWorking["2027-01-10"] {
		t.Error("a window that is only proposed must not remove a crushing day")
	}
	if nonWorking["2027-02-10"] {
		t.Error("a line outage must not remove a crushing day")
	}

	// Approving the proposal turns its days into non-working days.
	proposed.Status = domain.MaintenanceApproved
	proposed.ApprovedBy = "plant-manager"
	if _, err := exec.SaveMaintenance(ctx, proposed, "plant-manager"); err != nil {
		t.Fatalf("approve window: %v", err)
	}
	after, err := exec.ListMaintenance(ctx, store.ExecutionFilter{
		FactoryID: f.factory, Statuses: []string{domain.MaintenanceApproved},
	})
	must(t, err, "list approved maintenance after the approval")
	if days := domain.NonWorkingDays(after); len(days) != 5 {
		t.Errorf("non-working days after the approval = %d, want 5", len(days))
	}
	stale := approvedWindow
	approvedWindow.Description = "Boiler inspection and tube cleaning"
	if approvedWindow, err = exec.SaveMaintenance(ctx, approvedWindow, "engineering"); err != nil {
		t.Fatalf("amend window: %v", err)
	}
	if approvedWindow.RowVersion != 2 {
		t.Errorf("amended window is at version %d, want 2", approvedWindow.RowVersion)
	}
	if _, err := exec.SaveMaintenance(ctx, stale, "engineering"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("a stale maintenance window must conflict, got %v", err)
	}
}

// testBatches holds both stores to the same rules about lots: the code is
// unique across the system, and it is what somebody holding a pallet card looks
// a batch up by.
func testBatches(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)
	exec := s.Execution()

	saved, err := exec.SaveBatch(ctx, domain.Batch{
		Code: "B-2026-12-05-A", ProductID: f.product, FactoryID: f.factory,
		ProducedOn: "2026-12-05", Quantity: domain.D("330"),
	}, "shift")
	must(t, err, "save batch")
	if saved.ID == "" || saved.Status != "OPEN" {
		t.Fatalf("a new batch is open and has an id: %+v", saved)
	}

	byCode, err := exec.BatchByCode(ctx, "B-2026-12-05-A")
	must(t, err, "batch by code")
	if byCode.ID != saved.ID || !byCode.Quantity.Equal(domain.D("330")) {
		t.Errorf("the code must find the same batch, got %+v", byCode)
	}

	// The code is the business key, and two lots with the same code could not be
	// told apart on a certificate.
	if _, err := exec.SaveBatch(ctx, domain.Batch{
		Code: "B-2026-12-05-A", ProductID: f.product, FactoryID: f.factory,
	}, "shift"); !errors.Is(err, domain.ErrDuplicate) {
		t.Errorf("a duplicate batch code must be refused, got %v", err)
	}

	// A code nobody has used is not found rather than empty, so a caller cannot
	// mistake "no such batch" for "a batch with no id".
	if _, err := exec.BatchByCode(ctx, "B-NEVER-MADE"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("an unknown code must be not found, got %v", err)
	}
	if _, err := exec.GetBatch(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("a malformed id must be not found rather than an error, got %v", err)
	}

	saved.Quantity, saved.Status = domain.D("300"), "RELEASED"
	updated, err := exec.SaveBatch(ctx, saved, "supervisor")
	must(t, err, "update batch")
	if updated.RowVersion != saved.RowVersion+1 || updated.Status != "RELEASED" {
		t.Errorf("the update must bump the row version and stick: %+v", updated)
	}

	page, err := exec.ListBatches(ctx, store.ExecutionFilter{FactoryID: f.factory})
	must(t, err, "list batches")
	if len(page.Items) != 1 {
		t.Fatalf("one batch, got %d", len(page.Items))
	}
	page, err = exec.ListBatches(ctx, store.ExecutionFilter{Number: "B-2026-12-05-A"})
	must(t, err, "list by code")
	if len(page.Items) != 1 {
		t.Errorf("filtering by code must find it, got %d", len(page.Items))
	}
	page, err = exec.ListBatches(ctx, store.ExecutionFilter{Statuses: []string{"OPEN"}})
	must(t, err, "list by status")
	if len(page.Items) != 0 {
		t.Errorf("the batch is released, so no open batch matches, got %d", len(page.Items))
	}
}
