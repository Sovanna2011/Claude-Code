// Package storetest holds the conformance suite that every store
// implementation must pass.
//
// The in-memory store and the PostgreSQL store are used interchangeably by the
// service layer, so they have to behave identically: same business keys, same
// optimistic concurrency, same upsert semantics, same transaction rollback.
// Running one suite against both is what keeps that promise honest.
package storetest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Factory creates a fresh, empty store for one test.
type Factory func(t *testing.T) store.Store

// Run executes the whole conformance suite.
func Run(t *testing.T, newStore Factory) {
	t.Run("MasterDataLifecycle", func(t *testing.T) { testMasterData(t, newStore) })
	t.Run("MasterDataConcurrency", func(t *testing.T) { testConcurrency(t, newStore) })
	t.Run("Paging", func(t *testing.T) { testPaging(t, newStore) })
	t.Run("SeasonsAndVersions", func(t *testing.T) { testSeasons(t, newStore) })
	t.Run("AssumptionsAndMix", func(t *testing.T) { testAssumptions(t, newStore) })
	t.Run("DailyFactUpsert", func(t *testing.T) { testDailyFacts(t, newStore) })
	t.Run("TransactionRollback", func(t *testing.T) { testTransaction(t, newStore) })
	t.Run("Audit", func(t *testing.T) { testAudit(t, newStore) })
	t.Run("Idempotency", func(t *testing.T) { testIdempotency(t, newStore) })
	t.Run("ProductionOrders", func(t *testing.T) { testOrders(t, newStore) })
	t.Run("Confirmations", func(t *testing.T) { testConfirmations(t, newStore) })
	t.Run("InventoryPosting", func(t *testing.T) { testInventory(t, newStore) })
	t.Run("InventoryRollback", func(t *testing.T) { testInventoryRollback(t, newStore) })
	t.Run("Quality", func(t *testing.T) { testQuality(t, newStore) })
	t.Run("Maintenance", func(t *testing.T) { testMaintenance(t, newStore) })
	t.Run("CostElements", func(t *testing.T) { testCostElements(t, newStore) })
	t.Run("CostRates", func(t *testing.T) { testCostRates(t, newStore) })
	t.Run("ExchangeRates", func(t *testing.T) { testExchangeRates(t, newStore) })
	t.Run("CostRuns", func(t *testing.T) { testCostRuns(t, newStore) })
}

// fixture seeds the minimum master data the planning tests need and returns
// the ids.
type fixture struct {
	company, factory, warehouse, product, channel, customer string
}

func seedFixture(t *testing.T, ctx context.Context, s store.Store) fixture {
	t.Helper()
	md := s.MasterData()

	company, err := md.Companies().Save(ctx, domain.Company{
		Code: "KSS", Name: "Kampong Speu Sugar Co., Ltd.", Currency: "USD",
		TimeZone: "Asia/Phnom_Penh", Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save company")

	factory, err := md.Factories().Save(ctx, domain.Factory{
		CompanyID: company.ID, Code: "F1", Name: "Factory 1",
		TimeZone: "Asia/Phnom_Penh", Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save factory")

	_, err = md.ProductCategories().Save(ctx, domain.ProductCategory{
		Code: "SUGAR", Name: "Sugar", Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save category")

	_, err = md.UOMs().Save(ctx, domain.UnitOfMeasure{
		Code: "TON", Name: "Metric ton", Dimension: "MASS", Decimals: 3,
		Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save uom")

	product, err := md.Products().Save(ctx, domain.Product{
		Code: "REF", Name: "Refined Sugar", CategoryCode: "SUGAR", BaseUOM: "TON",
		Stage: domain.StageRefining, StorageClass: domain.StorageFinished, IsFinished: true,
		Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save product")

	warehouse, err := md.Warehouses().Save(ctx, domain.Warehouse{
		FactoryID: factory.ID, Code: "FG-WH1", Name: "Finished Goods Warehouse 1",
		StorageClass: domain.StorageFinished, CapacityTons: domain.D("22000"),
		UsablePct: domain.D("100"), Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save warehouse")

	customer, err := md.Customers().Save(ctx, domain.Customer{
		Code: "C1", Name: "Domestic Trader", Country: "KH", Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save customer")

	channel, err := md.Channels().Save(ctx, domain.ShipmentChannel{
		Code: "QUOTA", Name: "Quota shipment", CustomerID: customer.ID, Category: "QUOTA",
		Validity: domain.Validity{Active: true},
	}, "seed")
	must(t, err, "save channel")

	return fixture{company.ID, factory.ID, warehouse.ID, product.ID, channel.ID, customer.ID}
}

func must(t *testing.T, err error, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

// ---------------------------------------------------------------------------

func testMasterData(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)
	repo := s.MasterData().Warehouses()

	got, err := repo.Get(ctx, f.warehouse)
	must(t, err, "get warehouse")
	if got.Code != "FG-WH1" || got.RowVersion != 1 {
		t.Fatalf("unexpected warehouse %+v", got)
	}
	if got.CreatedBy != "seed" || got.CreatedAt.IsZero() {
		t.Error("audit fields must be populated on insert")
	}

	byCode, err := repo.GetByCode(ctx, "FG-WH1")
	must(t, err, "get warehouse by code")
	if byCode.ID != got.ID {
		t.Error("GetByCode must resolve the same record")
	}

	// Update bumps the row version and keeps the creation audit.
	got.Name = "Finished Goods Warehouse 1 (renamed)"
	updated, err := repo.Save(ctx, got, "planner")
	must(t, err, "update warehouse")
	if updated.RowVersion != 2 {
		t.Errorf("row version = %d, want 2", updated.RowVersion)
	}
	if updated.CreatedBy != "seed" || updated.UpdatedBy != "planner" {
		t.Errorf("audit fields wrong after update: created by %s, updated by %s",
			updated.CreatedBy, updated.UpdatedBy)
	}

	// A duplicate business key is rejected.
	_, err = repo.Save(ctx, domain.Warehouse{
		FactoryID: f.factory, Code: "FG-WH1", Name: "Duplicate",
		StorageClass: domain.StorageFinished, CapacityTons: domain.D("100"),
		UsablePct: domain.D("100"), Validity: domain.Validity{Active: true},
	}, "planner")
	if !errors.Is(err, domain.ErrDuplicate) {
		t.Errorf("duplicate code error = %v, want ErrDuplicate", err)
	}

	// Missing records are not found, not empty results.
	if _, err := repo.Get(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("missing warehouse error = %v, want ErrNotFound", err)
	}

	// Soft delete keeps the row and hides it from the active list.
	must(t, repo.Deactivate(ctx, updated.ID, updated.RowVersion, "admin"), "deactivate")
	after, err := repo.Get(ctx, updated.ID)
	must(t, err, "get after deactivate")
	if after.Active {
		t.Error("the warehouse must be inactive after deactivation")
	}
	active := true
	page, err := repo.List(ctx, store.ListOptions{Active: &active})
	must(t, err, "list active")
	for _, w := range page.Items {
		if w.ID == updated.ID {
			t.Error("a deactivated warehouse must not appear in the active list")
		}
	}
}

func testConcurrency(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	seedFixture(t, ctx, s)
	repo := s.MasterData().Companies()

	first, err := repo.GetByCode(ctx, "KSS")
	must(t, err, "get company")
	second := first // two users opened the same record

	first.Name = "Changed by user one"
	if _, err := repo.Save(ctx, first, "user-one"); err != nil {
		t.Fatalf("first save: %v", err)
	}
	second.Name = "Changed by user two"
	_, err = repo.Save(ctx, second, "user-two")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale save error = %v, want ErrConflict", err)
	}
	// The message has to tell the second user what happened.
	if err != nil && !contains(err.Error(), "user-one") {
		t.Errorf("conflict message should name the other editor: %v", err)
	}

	current, err := repo.GetByCode(ctx, "KSS")
	must(t, err, "re-read company")
	if current.Name != "Changed by user one" {
		t.Errorf("the losing write must not have been applied, got %q", current.Name)
	}
}

func testPaging(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	seedFixture(t, ctx, s)
	repo := s.MasterData().ReasonCodes()

	for _, c := range []struct{ code, name, category string }{
		{"DT-01", "Boiler stop", "DOWNTIME"},
		{"DT-02", "Mill jam", "DOWNTIME"},
		{"DT-03", "Power failure", "DOWNTIME"},
		{"LS-01", "Handling loss", "LOSS"},
		{"VR-01", "Cane quality", "VARIANCE"},
	} {
		_, err := repo.Save(ctx, domain.ReasonCode{
			Code: c.code, Name: c.name, Category: c.category, Validity: domain.Validity{Active: true},
		}, "seed")
		must(t, err, "save reason code "+c.code)
	}

	page, err := repo.List(ctx, store.ListOptions{Top: 2})
	must(t, err, "list page 1")
	if len(page.Items) != 2 || page.Count != 5 {
		t.Fatalf("page 1 = %d items of %d, want 2 of 5", len(page.Items), page.Count)
	}
	if page.Items[0].Code != "DT-01" {
		t.Errorf("results must be ordered by code, got %s first", page.Items[0].Code)
	}

	page2, err := repo.List(ctx, store.ListOptions{Top: 2, Skip: 2})
	must(t, err, "list page 2")
	if len(page2.Items) != 2 || page2.Items[0].Code != "DT-03" {
		t.Errorf("page 2 = %+v", codes(page2.Items))
	}

	// Skipping past the end is an empty page, not an error.
	empty, err := repo.List(ctx, store.ListOptions{Top: 10, Skip: 99})
	must(t, err, "list past the end")
	if len(empty.Items) != 0 || empty.Count != 5 {
		t.Errorf("past-the-end page = %d items of %d", len(empty.Items), empty.Count)
	}

	// Search is case insensitive and matches code or name.
	found, err := repo.List(ctx, store.ListOptions{Search: "power"})
	must(t, err, "search")
	if len(found.Items) != 1 || found.Items[0].Code != "DT-03" {
		t.Errorf("search returned %v", codes(found.Items))
	}

	// Parent filter narrows by the entity's natural parent.
	downtime, err := repo.List(ctx, store.ListOptions{ParentID: "DOWNTIME"})
	must(t, err, "filter by category")
	if len(downtime.Items) != 3 {
		t.Errorf("downtime reasons = %v, want 3", codes(downtime.Items))
	}
}

func testSeasons(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)
	p := s.Planning()

	season, err := p.SaveSeason(ctx, domain.Season{
		CompanyID: f.company, FactoryID: f.factory, Code: "2026-2027", Name: "Season 2026-2027",
		StartDate: "2026-12-01", EndDate: "2027-04-16", PlannedDays: 137, Status: "OPEN",
	}, "planner")
	must(t, err, "save season")
	if season.ID == "" || season.RowVersion != 1 {
		t.Fatalf("unexpected season %+v", season)
	}

	got, err := p.GetSeason(ctx, season.ID)
	must(t, err, "get season")
	if got.StartDate != "2026-12-01" || got.PlannedDays != 137 {
		t.Errorf("season round trip lost data: %+v", got)
	}

	// Version numbers are assigned per season.
	v1, err := p.SaveVersion(ctx, domain.PlanVersion{
		SeasonID: season.ID, Code: "V1", Description: "Original budget",
		PlanType: domain.PlanTypeBudget, Status: domain.StatusDraft, Owner: "planner",
	}, "planner")
	must(t, err, "save version 1")
	if v1.VersionNo != 1 {
		t.Errorf("first version number = %d, want 1", v1.VersionNo)
	}

	v2, err := p.SaveVersion(ctx, domain.PlanVersion{
		SeasonID: season.ID, Code: "V2", Description: "Revised",
		PlanType: domain.PlanTypeRevised, Status: domain.StatusDraft, Owner: "planner",
	}, "planner")
	must(t, err, "save version 2")
	if v2.VersionNo != 2 {
		t.Errorf("second version number = %d, want 2", v2.VersionNo)
	}

	// A duplicate code inside one season is rejected.
	_, err = p.SaveVersion(ctx, domain.PlanVersion{
		SeasonID: season.ID, Code: "V1", PlanType: domain.PlanTypeForecast,
		Status: domain.StatusDraft,
	}, "planner")
	if !errors.Is(err, domain.ErrDuplicate) {
		t.Errorf("duplicate version code error = %v, want ErrDuplicate", err)
	}

	list, err := p.ListVersions(ctx, season.ID, store.ListOptions{})
	must(t, err, "list versions")
	if list.Count != 2 {
		t.Errorf("versions = %d, want 2", list.Count)
	}

	// Status transitions round trip, including the nullable timestamps.
	v1.Status = domain.StatusReleased
	v1.LockedThrough = "2027-01-31"
	released, err := p.SaveVersion(ctx, v1, "approver")
	must(t, err, "release version")
	reread, err := p.GetVersion(ctx, released.ID)
	must(t, err, "re-read version")
	if reread.Status != domain.StatusReleased || reread.LockedThrough != "2027-01-31" {
		t.Errorf("version state lost: %+v", reread)
	}

	// Stale writes are rejected here too.
	v1.RowVersion = 1
	if _, err := p.SaveVersion(ctx, v1, "planner"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("stale version save error = %v, want ErrConflict", err)
	}
}

func testAssumptions(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)
	p := s.Planning()

	season, err := p.SaveSeason(ctx, domain.Season{
		CompanyID: f.company, FactoryID: f.factory, Code: "2026-2027", Name: "Season",
		StartDate: "2026-12-01", PlannedDays: 137, Status: "OPEN",
	}, "planner")
	must(t, err, "save season")
	v, err := p.SaveVersion(ctx, domain.PlanVersion{
		SeasonID: season.ID, Code: "V1", PlanType: domain.PlanTypeBudget, Status: domain.StatusDraft,
	}, "planner")
	must(t, err, "save version")

	a, err := p.SaveAssumption(ctx, domain.PlanAssumption{
		VersionID: v.ID, Code: domain.AsmRecoveryPct, Description: "Raw sugar recovery",
		Value: domain.D("11.00"), UOM: "%",
	}, "planner")
	must(t, err, "save assumption")

	// Saving the same business key again updates rather than duplicating: an
	// import can be re-run safely.
	a2, err := p.SaveAssumption(ctx, domain.PlanAssumption{
		VersionID: v.ID, Code: domain.AsmRecoveryPct, Description: "Raw sugar recovery",
		Value: domain.D("11.50"), UOM: "%",
	}, "planner")
	must(t, err, "re-save assumption")
	if a2.ID != a.ID {
		t.Error("the same assumption key must update the same row")
	}

	list, err := p.ListAssumptions(ctx, v.ID)
	must(t, err, "list assumptions")
	if len(list) != 1 {
		t.Fatalf("assumptions = %d, want 1", len(list))
	}
	if !list[0].Value.Equal(domain.D("11.50")) {
		t.Errorf("assumption value = %s, want 11.50", list[0].Value)
	}
	if list[0].RowVersion < 2 {
		t.Errorf("re-saving must bump the row version, got %d", list[0].RowVersion)
	}

	// Product mix behaves the same way.
	m, err := p.SaveMix(ctx, domain.ProductMixEntry{
		VersionID: v.ID, ProductID: f.product, WarehouseID: f.warehouse,
		SeasonTons: domain.D("106700"),
	}, "planner")
	must(t, err, "save mix")
	m2, err := p.SaveMix(ctx, domain.ProductMixEntry{
		VersionID: v.ID, ProductID: f.product, WarehouseID: f.warehouse,
		SeasonTons: domain.D("110000"),
	}, "planner")
	must(t, err, "re-save mix")
	if m2.ID != m.ID {
		t.Error("the same mix key must update the same row")
	}
	mixes, err := p.ListMix(ctx, v.ID)
	must(t, err, "list mix")
	if len(mixes) != 1 || !mixes[0].SeasonTons.Equal(domain.D("110000")) {
		t.Errorf("mix = %+v", mixes)
	}

	must(t, p.DeleteMix(ctx, m.ID), "delete mix")
	must(t, p.DeleteAssumption(ctx, a.ID), "delete assumption")
	if err := p.DeleteAssumption(ctx, a.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("deleting twice = %v, want ErrNotFound", err)
	}
}

func testDailyFacts(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)
	p := s.Planning()

	season, err := p.SaveSeason(ctx, domain.Season{
		CompanyID: f.company, FactoryID: f.factory, Code: "2026-2027", Name: "Season",
		StartDate: "2026-12-01", PlannedDays: 137, Status: "OPEN",
	}, "planner")
	must(t, err, "save season")
	v, err := p.SaveVersion(ctx, domain.PlanVersion{
		SeasonID: season.ID, Code: "V1", PlanType: domain.PlanTypeBudget, Status: domain.StatusDraft,
	}, "planner")
	must(t, err, "save version")

	cane := []domain.DailyCanePlan{
		{VersionID: v.ID, FactoryID: f.factory, BusinessDate: "2026-12-01", Series: domain.SeriesPlan,
			CaneCrushed: domain.D("16788.321"), CrushRateTPH: domain.D("700"), AvailableHrs: domain.D("24")},
		{VersionID: v.ID, FactoryID: f.factory, BusinessDate: "2026-12-02", Series: domain.SeriesPlan,
			CaneCrushed: domain.D("16788.321"), CrushRateTPH: domain.D("700"), AvailableHrs: domain.D("24")},
	}
	n, err := p.UpsertCane(ctx, cane, "planner")
	must(t, err, "upsert cane")
	if n != 2 {
		t.Errorf("upserted %d rows, want 2", n)
	}

	// Re-running the same upsert replaces the rows instead of duplicating them.
	cane[0].CaneCrushed = domain.D("17000")
	_, err = p.UpsertCane(ctx, cane, "planner")
	must(t, err, "re-upsert cane")

	rows, err := p.ListCane(ctx, store.PlanFilter{VersionIDs: []string{v.ID}})
	must(t, err, "list cane")
	if len(rows) != 2 {
		t.Fatalf("cane rows = %d, want 2 after a repeated upsert", len(rows))
	}
	if !rows[0].CaneCrushed.Equal(domain.D("17000")) {
		t.Errorf("the second upsert must win: %s", rows[0].CaneCrushed)
	}
	if rows[0].RowVersion < 2 {
		t.Errorf("re-upsert must bump the row version, got %d", rows[0].RowVersion)
	}

	// The decimal survives the round trip exactly.
	if !rows[1].CaneCrushed.Equal(domain.D("16788.321")) {
		t.Errorf("decimal round trip lost precision: %s", rows[1].CaneCrushed)
	}

	// Date range filtering.
	ranged, err := p.ListCane(ctx, store.PlanFilter{
		VersionIDs: []string{v.ID}, From: "2026-12-02", To: "2026-12-02",
	})
	must(t, err, "list cane in range")
	if len(ranged) != 1 || ranged[0].BusinessDate != "2026-12-02" {
		t.Errorf("date range filter returned %d rows", len(ranged))
	}

	// Plan and actual rows for the same date coexist: an actual never
	// overwrites the plan.
	_, err = p.UpsertCane(ctx, []domain.DailyCanePlan{{
		VersionID: v.ID, FactoryID: f.factory, BusinessDate: "2026-12-01",
		Series: domain.SeriesActual, CaneCrushed: domain.D("15000"), AvailableHrs: domain.D("24"),
	}}, "operator")
	must(t, err, "upsert actual cane")
	all, err := p.ListCane(ctx, store.PlanFilter{VersionIDs: []string{v.ID}})
	must(t, err, "list all cane")
	if len(all) != 3 {
		t.Errorf("cane rows = %d, want 3 (2 plan + 1 actual)", len(all))
	}
	planOnly, err := p.ListCane(ctx, store.PlanFilter{VersionIDs: []string{v.ID}, Series: domain.SeriesPlan})
	must(t, err, "list plan cane")
	if len(planOnly) != 2 {
		t.Errorf("plan rows = %d, want 2", len(planOnly))
	}

	// Storage, product and shipment rows behave the same way.
	_, err = p.UpsertStorage(ctx, []domain.DailyStoragePlan{{
		VersionID: v.ID, WarehouseID: f.warehouse, ProductID: f.product,
		BusinessDate: "2026-12-01", Series: domain.SeriesPlan,
		ProductionReceipt: domain.D("1767.153"), ShipmentQty: domain.D("500"),
		EndingBalance: domain.D("1267.153"),
	}}, "planner")
	must(t, err, "upsert storage")

	_, err = p.UpsertProducts(ctx, []domain.DailyProductPlan{{
		VersionID: v.ID, FactoryID: f.factory, BusinessDate: "2026-12-01", ProductID: f.product,
		Series: domain.SeriesPlan, Quantity: domain.D("778.832"), RemeltInput: domain.D("817.774"),
	}}, "planner")
	must(t, err, "upsert products")

	_, err = p.UpsertShipments(ctx, []domain.DailyShipmentPlan{{
		VersionID: v.ID, WarehouseID: f.warehouse, ProductID: f.product, ChannelID: f.channel,
		BusinessDate: "2026-12-01", Series: domain.SeriesPlan, Quantity: domain.D("500"),
	}}, "planner")
	must(t, err, "upsert shipments")

	storage, err := p.ListStorage(ctx, store.PlanFilter{VersionIDs: []string{v.ID}})
	must(t, err, "list storage")
	if len(storage) != 1 || !storage[0].ProductionReceipt.Equal(domain.D("1767.153")) {
		t.Errorf("storage round trip: %+v", storage)
	}
	prods, err := p.ListProducts(ctx, store.PlanFilter{VersionIDs: []string{v.ID}})
	must(t, err, "list products")
	if len(prods) != 1 {
		t.Errorf("product rows = %d, want 1", len(prods))
	}
	ships, err := p.ListShipments(ctx, store.PlanFilter{VersionIDs: []string{v.ID}})
	must(t, err, "list shipments")
	if len(ships) != 1 {
		t.Errorf("shipment rows = %d, want 1", len(ships))
	}

	// Regenerating a plan clears every daily row of the version.
	must(t, p.DeleteVersionRows(ctx, v.ID), "delete version rows")
	if rows, _ := p.ListCane(ctx, store.PlanFilter{VersionIDs: []string{v.ID}}); len(rows) != 0 {
		t.Errorf("cane rows remain after delete: %d", len(rows))
	}
	if rows, _ := p.ListStorage(ctx, store.PlanFilter{VersionIDs: []string{v.ID}}); len(rows) != 0 {
		t.Errorf("storage rows remain after delete: %d", len(rows))
	}
	if rows, _ := p.ListProducts(ctx, store.PlanFilter{VersionIDs: []string{v.ID}}); len(rows) != 0 {
		t.Errorf("product rows remain after delete: %d", len(rows))
	}
	if rows, _ := p.ListShipments(ctx, store.PlanFilter{VersionIDs: []string{v.ID}}); len(rows) != 0 {
		t.Errorf("shipment rows remain after delete: %d", len(rows))
	}
}

// testTransaction is the requirement from section 24: a failure part way
// through a posting must leave nothing behind.
func testTransaction(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)
	p := s.Planning()

	season, err := p.SaveSeason(ctx, domain.Season{
		CompanyID: f.company, FactoryID: f.factory, Code: "2026-2027", Name: "Season",
		StartDate: "2026-12-01", PlannedDays: 137, Status: "OPEN",
	}, "planner")
	must(t, err, "save season")

	boom := errors.New("posting step failed")
	err = s.InTx(ctx, func(tx store.Store) error {
		v, err := tx.Planning().SaveVersion(ctx, domain.PlanVersion{
			SeasonID: season.ID, Code: "V-ROLLBACK", PlanType: domain.PlanTypeBudget,
			Status: domain.StatusDraft,
		}, "planner")
		if err != nil {
			return err
		}
		if _, err := tx.Planning().UpsertCane(ctx, []domain.DailyCanePlan{{
			VersionID: v.ID, FactoryID: f.factory, BusinessDate: "2026-12-01",
			Series: domain.SeriesPlan, CaneCrushed: domain.D("16788"), AvailableHrs: domain.D("24"),
		}}, "planner"); err != nil {
			return err
		}
		// Everything above succeeded; now the posting fails.
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("InTx error = %v, want the posting error", err)
	}

	list, err := p.ListVersions(ctx, season.ID, store.ListOptions{})
	must(t, err, "list versions after rollback")
	if list.Count != 0 {
		t.Errorf("the rolled back version is still there: %d versions", list.Count)
	}

	// A successful transaction commits everything.
	err = s.InTx(ctx, func(tx store.Store) error {
		v, err := tx.Planning().SaveVersion(ctx, domain.PlanVersion{
			SeasonID: season.ID, Code: "V-COMMIT", PlanType: domain.PlanTypeBudget,
			Status: domain.StatusDraft,
		}, "planner")
		if err != nil {
			return err
		}
		_, err = tx.Planning().UpsertCane(ctx, []domain.DailyCanePlan{{
			VersionID: v.ID, FactoryID: f.factory, BusinessDate: "2026-12-01",
			Series: domain.SeriesPlan, CaneCrushed: domain.D("16788"), AvailableHrs: domain.D("24"),
		}}, "planner")
		return err
	})
	must(t, err, "committing transaction")

	list, err = p.ListVersions(ctx, season.ID, store.ListOptions{})
	must(t, err, "list versions after commit")
	if list.Count != 1 || list.Items[0].Code != "V-COMMIT" {
		t.Errorf("expected exactly the committed version, got %d", list.Count)
	}
	rows, err := p.ListCane(ctx, store.PlanFilter{VersionIDs: []string{list.Items[0].ID}})
	must(t, err, "list cane after commit")
	if len(rows) != 1 {
		t.Errorf("committed cane rows = %d, want 1", len(rows))
	}
}

func testAudit(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	a := s.Audit()

	must(t, a.Append(ctx, domain.AuditEvent{
		Actor: "planner", Action: "RELEASE", Entity: "plan_version", EntityID: "v-1",
		Reason: "season start", CorrelationID: "req-1",
		Before: `{"status":"APPROVED"}`, After: `{"status":"RELEASED"}`,
	}), "append audit")
	must(t, a.Append(ctx, domain.AuditEvent{
		Actor: "operator", Action: "POST", Entity: "daily_cane_plan", EntityID: "c-1",
		CorrelationID: "req-2",
	}), "append audit 2")

	all, err := a.List(ctx, store.AuditFilter{})
	must(t, err, "list audit")
	if all.Count != 2 {
		t.Fatalf("audit events = %d, want 2", all.Count)
	}

	filtered, err := a.List(ctx, store.AuditFilter{Entity: "plan_version"})
	must(t, err, "filter audit")
	if filtered.Count != 1 || filtered.Items[0].Action != "RELEASE" {
		t.Errorf("filtered audit = %+v", filtered.Items)
	}
	if filtered.Items[0].Before == "" || filtered.Items[0].After == "" {
		t.Error("before and after states must survive the round trip")
	}

	byActor, err := a.List(ctx, store.AuditFilter{Actor: "operator"})
	must(t, err, "filter audit by actor")
	if byActor.Count != 1 {
		t.Errorf("events for operator = %d, want 1", byActor.Count)
	}
}

func testIdempotency(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	idem := s.Idempotency()

	fresh, prev, err := idem.Remember(ctx, "key-1", "/api/v1/plans/import", []byte(`{"imported":42}`))
	must(t, err, "remember key")
	if !fresh || prev != nil {
		t.Fatalf("first call: fresh=%v previous=%s", fresh, prev)
	}

	fresh, prev, err = idem.Remember(ctx, "key-1", "/api/v1/plans/import", []byte(`{"imported":42}`))
	must(t, err, "remember key again")
	if fresh {
		t.Error("a repeated key must not be treated as fresh")
	}
	if string(prev) != `{"imported":42}` {
		t.Errorf("the first response must be replayed, got %s", prev)
	}

	// The same key on a different endpoint is a different request.
	fresh, _, err = idem.Remember(ctx, "key-1", "/api/v1/plans/release", nil)
	must(t, err, "remember key on another endpoint")
	if !fresh {
		t.Error("the key is scoped to the endpoint")
	}

	// A posting claims its key before doing the work and attaches the response
	// afterwards, so a retry replays the document rather than an acknowledgement.
	fresh, _, err = idem.Remember(ctx, "key-2", "/api/v1/inventory/documents", nil)
	must(t, err, "claim a posting key")
	if !fresh {
		t.Fatal("a new key must be fresh")
	}
	must(t, idem.Complete(ctx, "key-2", "/api/v1/inventory/documents", []byte(`{"documentNo":"MD-1"}`)),
		"complete the key")

	_, prev, err = idem.Remember(ctx, "key-2", "/api/v1/inventory/documents", nil)
	must(t, err, "retry the posting")
	if string(prev) != `{"documentNo":"MD-1"}` {
		t.Errorf("the retry must replay the first response, got %s", prev)
	}

	// Completing a key nobody claimed would defeat the claim, so it is refused.
	if err := idem.Complete(ctx, "never-claimed", "/api/v1/inventory/documents", nil); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("completing an unclaimed key must be not found, got %v", err)
	}
}

func codes[T any](items []T) []string {
	out := make([]string, 0, len(items))
	for _, i := range items {
		switch v := any(i).(type) {
		case domain.ReasonCode:
			out = append(out, v.Code)
		default:
			out = append(out, "?")
		}
	}
	return out
}

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }
