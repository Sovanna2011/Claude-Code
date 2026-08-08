package domain

import "testing"

// referenceInput builds the Kampong Speu 2026-2027 scenario from section 2 of
// the specification. The same figures are used by the seed data, so a change to
// one is caught here.
func referenceInput() GeneratorInput {
	rawWH1 := Warehouse{ID: "wh-raw-1", Code: "RAW-WH1", Name: "Raw Sugar Warehouse 1",
		StorageClass: StorageRaw, CapacityTons: D("45000"), UsablePct: D("100")}
	rawWH2 := Warehouse{ID: "wh-raw-2", Code: "RAW-WH2", Name: "Raw Sugar Warehouse 2",
		StorageClass: StorageRaw, CapacityTons: D("65000"), UsablePct: D("100")}
	fgWH1 := Warehouse{ID: "wh-fg-1", Code: "FG-WH1", Name: "Finished Goods Warehouse 1",
		StorageClass: StorageFinished, CapacityTons: D("22000"), UsablePct: D("100")}
	fgWH3 := Warehouse{ID: "wh-fg-3", Code: "FG-WH3", Name: "Finished Goods Warehouse 3",
		StorageClass: StorageFinished, CapacityTons: D("47000"), UsablePct: D("100")}

	return GeneratorInput{
		Season: Season{ID: "season-1", FactoryID: "fac-1", Code: "2026-2027",
			StartDate: "2026-12-01", PlannedDays: 137},
		Version: PlanVersion{ID: "ver-1", Code: "V1", PlanType: PlanTypeBudget, Status: StatusDraft},
		Assumptions: map[string]Dec{
			AsmCaneTarget:        D("2300000"),
			AsmSeasonDays:        D("137"),
			AsmRecoveryPct:       D("11.00"),
			AsmDirectToRefinePct: D("49.387"), // 124,950 / 253,000
			AsmRemeltInputFactor: D("1.05"),
			AsmQuotaShipmentTPD:  D("500"),
			AsmCrushRateTPH:      D("700"),
			AsmAvailableHours:    D("24"),
			AsmCapacityWarnPct:   D("80"),
			AsmCapacityAlertPct:  D("90"),
		},
		Mix: []ProductMixEntry{
			{ID: "mix-1", ProductID: "p-refined", SeasonTons: D("106700"), WarehouseID: "wh-fg-3"},
			{ID: "mix-2", ProductID: "p-white", SeasonTons: D("133400"), WarehouseID: "wh-fg-1"},
			{ID: "mix-3", ProductID: "p-super", SeasonTons: D("2000"), WarehouseID: "wh-fg-1"},
		},
		Products: map[string]Product{
			"p-refined": {ID: "p-refined", Code: "REF", Name: "Refined Sugar"},
			"p-white":   {ID: "p-white", Code: "WHT", Name: "White Sugar"},
			"p-super":   {ID: "p-super", Code: "SUP", Name: "Super Refined Sugar"},
			"p-raw":     {ID: "p-raw", Code: "RAW", Name: "Raw Sugar"},
		},
		Warehouses: map[string]Warehouse{
			"wh-raw-1": rawWH1, "wh-raw-2": rawWH2, "wh-fg-1": fgWH1, "wh-fg-3": fgWH3,
		},
		RawWarehouseIDs: []string{"wh-raw-1", "wh-raw-2"},
		RawProductID:    "p-raw",
		QuotaChannelID:  "ch-quota",
	}
}

func TestGenerateReferenceScenario(t *testing.T) {
	out, err := Generate(referenceInput())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(out.Dates) != 137 {
		t.Fatalf("season length = %d days, want 137", len(out.Dates))
	}
	if out.Dates[0] != "2026-12-01" {
		t.Errorf("first day = %s", out.Dates[0])
	}
	// 137 days from 1 December 2026 ends on 16 April 2027.
	if got := out.Dates[136]; got != "2027-04-16" {
		t.Errorf("last day = %s, want 2027-04-16", got)
	}

	// The daily allocation must add back to the season target exactly.
	eq(t, out.Summary.CaneAllocated, D("2300000"), "allocated cane")
	eq(t, out.Summary.RawSugarExpected, D("253000.000"), "expected raw sugar")
	eq(t, out.Summary.FinishedGoods, D("242100"), "finished goods")

	// The direct/storage split reproduces the workbook figures to the ton.
	if diff := out.Summary.RawDirectRefine.Sub(D("124950")).Abs(); diff.GreaterThan(D("1")) {
		t.Errorf("raw sugar direct to refining = %s, want about 124,950", out.Summary.RawDirectRefine)
	}
	if diff := out.Summary.RawToStorage.Sub(D("128050")).Abs(); diff.GreaterThan(D("1")) {
		t.Errorf("raw sugar to storage = %s, want about 128,050", out.Summary.RawToStorage)
	}
	eq(t, out.Summary.RawDirectRefine.Add(out.Summary.RawToStorage), out.Summary.RawSugarExpected,
		"the raw sugar split must be exhaustive")

	// Remelt input is finished goods times the effective-dated factor.
	eq(t, out.Summary.RemeltInput, RoundQty(D("242100").Mul(D("1.05"))), "remelt input at 1.05")

	if len(out.Cane) != 137 {
		t.Errorf("cane rows = %d, want 137", len(out.Cane))
	}
	if len(out.Products) != 137*3 {
		t.Errorf("product rows = %d, want %d", len(out.Products), 137*3)
	}
}

func TestGenerateLedgerContinuityForEveryDate(t *testing.T) {
	in := referenceInput()
	out, err := Generate(in)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Group the generated storage rows and roll each store forward. The
	// acceptance criterion is that continuity holds for every date and no
	// balance goes negative.
	byStore := map[string][]LedgerMovement{}
	for _, s := range out.Storage {
		key := s.WarehouseID + "|" + s.ProductID
		byStore[key] = append(byStore[key], LedgerMovement{
			Date:              s.BusinessDate,
			ProductionReceipt: s.ProductionReceipt,
			RemeltIssue:       s.RemeltIssue,
			ShipmentQty:       s.ShipmentQty,
		})
	}
	if len(byStore) == 0 {
		t.Fatal("no storage rows generated")
	}
	for key, movs := range byStore {
		days := RollLedger(Zero, D("100000"), movs)
		for i, d := range days {
			if i > 0 && !d.BeginningBalance.Equal(days[i-1].EndingBalance) {
				t.Fatalf("%s: continuity broken on %s", key, d.Date)
			}
			if d.EndingBalance.IsNegative() {
				t.Fatalf("%s: negative balance %s on %s", key, d.EndingBalance, d.Date)
			}
		}
	}
}

func TestGenerateRaisesCapacityWarning(t *testing.T) {
	in := referenceInput()
	out, err := Generate(in)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Raw sugar accumulates far beyond 110,000 t of storage when refining only
	// draws what it needs, so the generator must warn before the plan is
	// released rather than after the silos overflow.
	var found bool
	for _, w := range out.Warnings {
		if w.Entity == "warehouse" && w.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a capacity alert in the reference scenario, got %d warnings", len(out.Warnings))
	}
}

func TestGenerateFlagsRawSugarShortfallInTheReferencePlan(t *testing.T) {
	// The workbook plans 242,100 t of finished goods, which at the 1.05 remelt
	// factor needs 254,205 t of raw sugar, while the cane target and recovery
	// only yield 253,000 t. The plan is 1,205 t short and the generator has to
	// say so before anybody releases it.
	out, err := Generate(referenceInput())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var alert *Alert
	for i := range out.Warnings {
		if out.Warnings[i].Code == "REMELT_SUPPLY_SHORT" {
			alert = &out.Warnings[i]
		}
	}
	if alert == nil {
		t.Fatal("expected a raw sugar supply shortfall alert")
	}
	if alert.Severity != SeverityError {
		t.Errorf("shortfall severity = %s, want ERROR", alert.Severity)
	}
	eq(t, out.Summary.RemeltInput.Sub(out.Summary.RawSugarExpected), D("1205.000"), "shortfall")
}

func TestGenerateNeverDrawsRawStockBelowZero(t *testing.T) {
	in := referenceInput()
	// Refining demand far beyond supply must be capped at the stock on hand.
	in.Assumptions[AsmRemeltInputFactor] = D("3.0")
	out, err := Generate(in)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	stock := Zero
	for _, s := range out.Storage {
		if s.ProductID != in.RawProductID {
			continue
		}
		stock = stock.Add(s.ProductionReceipt).Sub(s.RemeltIssue)
		if stock.IsNegative() {
			t.Fatalf("raw stock went negative (%s) on %s", stock, s.BusinessDate)
		}
	}
}

func TestScaleSeriesPreservesTheRoundedTotal(t *testing.T) {
	daily := AllocateEvenly(D("2300000"), 137)
	scaled := ScaleSeries(daily, D("0.11"))
	eq(t, SumDec(scaled...), D("253000.000"), "cane x 11 % over 137 days")

	// The naive approach loses tons; this documents why ScaleSeries exists.
	naive := Zero
	for _, d := range daily {
		naive = naive.Add(RoundQty(d.Mul(D("0.11"))))
	}
	if naive.Equal(D("253000.000")) {
		t.Skip("rounding no longer drifts on this input")
	}
	if len(ScaleSeries(nil, D("2"))) != 0 {
		t.Error("an empty series scales to an empty series")
	}
}

func TestGenerateHonoursDailyRateCap(t *testing.T) {
	in := referenceInput()
	// Jumbo bag packing: 300 t/day up to 20,700 t, which is 69 days of work.
	in.Mix = append(in.Mix, ProductMixEntry{
		ID: "mix-jumbo", ProductID: "p-raw-jumbo", SeasonTons: D("20700"),
		DailyRateTons: D("300"), WarehouseID: "wh-fg-1",
	})
	out, err := Generate(in)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var days int
	total := Zero
	for _, p := range out.Products {
		if p.ProductID != "p-raw-jumbo" {
			continue
		}
		if p.Quantity.GreaterThan(D("300")) {
			t.Fatalf("daily jumbo output %s exceeds the 300 t/day rate on %s", p.Quantity, p.BusinessDate)
		}
		days++
		total = total.Add(p.Quantity)
	}
	if days != 69 {
		t.Errorf("jumbo packing spans %d days, want 69", days)
	}
	eq(t, total, D("20700"), "jumbo packing total")
}

func TestGenerateWarnsWhenRateCannotDeliverTonnage(t *testing.T) {
	in := referenceInput()
	in.Mix = []ProductMixEntry{{
		ID: "mix-slow", ProductID: "p-refined", SeasonTons: D("100000"),
		DailyRateTons: D("100"), WarehouseID: "wh-fg-1",
	}}
	out, err := Generate(in)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var found bool
	for _, w := range out.Warnings {
		if w.Code == "MIX_RATE_TOO_LOW" {
			found = true
		}
	}
	if !found {
		t.Error("expected a warning that 100 t/day cannot deliver 100,000 t in 137 days")
	}
}

func TestGenerateSkipsNonWorkingDays(t *testing.T) {
	in := referenceInput()
	in.Assumptions[AsmSeasonDays] = D("10")
	in.NonWorkingDays = map[BusinessDate]bool{"2026-12-03": true, "2026-12-04": true}
	out, err := Generate(in)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Ten crushing days, and the mill crushes on none of the maintenance days.
	// The assertion is on the cane rows rather than on out.Dates, because those
	// are two different things: out.Dates is the campaign calendar, which runs
	// continuously - the refinery and the shipping gate do not stop because the
	// mill is down.
	if out.Summary.WorkingDays != 10 {
		t.Fatalf("crushing days = %d, want 10", out.Summary.WorkingDays)
	}
	if len(out.Cane) != 10 {
		t.Fatalf("cane rows = %d, want 10", len(out.Cane))
	}
	for _, r := range out.Cane {
		if in.NonWorkingDays[r.BusinessDate] {
			t.Errorf("%s is a maintenance day and must not be crushed on", r.BusinessDate)
		}
	}
	// Ten working days starting 1 December, skipping the 3rd and 4th, run to
	// 12 December: the campaign is pushed out, the tonnage is not cut.
	if got := out.Cane[9].BusinessDate; got != "2026-12-12" {
		t.Errorf("last working day = %s, want 2026-12-12", got)
	}
	// And the campaign covers those twelve calendar days without a hole in it.
	if len(out.Dates) != 12 {
		t.Errorf("campaign = %d days, want the 12 calendar days 1-12 December", len(out.Dates))
	}
	for i := 1; i < len(out.Dates); i++ {
		if out.Dates[i] != out.Dates[i-1].AddDays(1) {
			t.Errorf("the campaign calendar jumps from %s to %s", out.Dates[i-1], out.Dates[i])
		}
	}
	eq(t, out.Summary.CaneAllocated, D("2300000"), "tonnage is preserved across a maintenance window")
}

func TestGenerateRejectsMissingAssumptions(t *testing.T) {
	in := referenceInput()
	delete(in.Assumptions, AsmRecoveryPct)
	_, err := Generate(in)
	if err == nil {
		t.Fatal("expected a validation error")
	}
	verr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected a ValidationError, got %T", err)
	}
	if len(verr.Errors) != 1 || verr.Errors[0].Code != "ASSUMPTION_MISSING" {
		t.Errorf("unexpected errors: %+v", verr.Errors)
	}
}

func TestGenerateRejectsOutOfRangeAssumptions(t *testing.T) {
	in := referenceInput()
	in.Assumptions[AsmRecoveryPct] = D("150")
	in.Assumptions[AsmSeasonDays] = D("0")
	_, err := Generate(in)
	verr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected a ValidationError, got %v", err)
	}
	if len(verr.Errors) != 2 {
		t.Errorf("expected two field errors, got %+v", verr.Errors)
	}
}

// ---------------------------------------------------------------------------
// Allocation helpers
// ---------------------------------------------------------------------------

func TestAllocateEvenlyPreservesTotal(t *testing.T) {
	cases := []struct {
		total string
		days  int
	}{
		{"2300000", 137}, {"253000", 137}, {"1", 3}, {"100", 7}, {"0", 5}, {"242100", 137},
	}
	for _, c := range cases {
		parts := AllocateEvenly(D(c.total), c.days)
		if len(parts) != c.days {
			t.Fatalf("AllocateEvenly(%s, %d) returned %d parts", c.total, c.days, len(parts))
		}
		eq(t, SumDec(parts...), RoundQty(D(c.total)), "sum of "+c.total+" allocation")
	}
	if got := AllocateEvenly(D("100"), 0); got != nil {
		t.Error("a zero-day season allocates nothing")
	}
}

func TestAllocateEvenlySpreadsTheRemainder(t *testing.T) {
	// 1 / 3 cannot be represented at three decimals; the parts must still sum
	// to exactly 1 and differ by at most one unit in the last place.
	parts := AllocateEvenly(D("1"), 3)
	eq(t, SumDec(parts...), D("1.000"), "sum")
	for _, p := range parts {
		if p.Sub(D("0.333")).Abs().GreaterThan(D("0.001")) {
			t.Errorf("part %s is not close to an even split", p)
		}
	}
}

func TestAllocateProportional(t *testing.T) {
	parts := AllocateProportional(D("1000"), []Dec{D("45000"), D("65000")})
	eq(t, SumDec(parts...), D("1000.000"), "sum")
	// 45/110 of 1,000 = 409.091
	eq(t, parts[0], D("409.091"), "first share")
	eq(t, parts[1], D("590.909"), "second share")

	even := AllocateProportional(D("100"), []Dec{Zero, Zero})
	eq(t, SumDec(even...), D("100.000"), "zero weights fall back to an even split")
	if got := AllocateProportional(D("100"), nil); got != nil {
		t.Error("no buckets, no allocation")
	}
}

func TestAllocateAtRate(t *testing.T) {
	parts, short := AllocateAtRate(D("20700"), D("300"), 137)
	eq(t, SumDec(parts...), D("20700"), "sum")
	eq(t, short, Zero, "no shortfall")
	var working int
	for _, p := range parts {
		if p.GreaterThan(Zero) {
			working++
		}
	}
	if working != 69 {
		t.Errorf("working days = %d, want 69", working)
	}

	_, short = AllocateAtRate(D("20700"), D("300"), 10)
	eq(t, short, D("17700"), "shortfall when the horizon is too short")

	even, _ := AllocateAtRate(D("100"), Zero, 4)
	eq(t, SumDec(even...), D("100.000"), "a zero rate falls back to an even split")
}

// ---------------------------------------------------------------------------
// Workflow
// ---------------------------------------------------------------------------

func TestPlanTransitions(t *testing.T) {
	cases := []struct {
		name    string
		status  PlanStatus
		typ     PlanType
		action  PlanAction
		reason  string
		wantTo  PlanStatus
		wantErr bool
	}{
		{"submit a draft", StatusDraft, PlanTypeBudget, ActionSubmit, "", StatusInReview, false},
		{"resubmit a rejected plan", StatusRejected, PlanTypeBudget, ActionSubmit, "", StatusInReview, false},
		{"recall from review", StatusInReview, PlanTypeBudget, ActionRecall, "", StatusDraft, false},
		{"approve", StatusInReview, PlanTypeBudget, ActionApprove, "", StatusApproved, false},
		{"reject needs a reason", StatusInReview, PlanTypeBudget, ActionReject, "", "", true},
		{"reject with a reason", StatusInReview, PlanTypeBudget, ActionReject, "cane target too high", StatusRejected, false},
		{"release an approved plan", StatusApproved, PlanTypeBudget, ActionRelease, "", StatusReleased, false},
		{"cannot release a draft", StatusDraft, PlanTypeBudget, ActionRelease, "", "", true},
		{"cannot approve a draft", StatusDraft, PlanTypeBudget, ActionApprove, "", "", true},
		{"reopen needs a reason", StatusReleased, PlanTypeBudget, ActionReopen, "", "", true},
		{"reopen with a reason", StatusReleased, PlanTypeBudget, ActionReopen, "revised cane forecast", StatusDraft, false},
		{"supersede a released plan", StatusReleased, PlanTypeBudget, ActionSupersede, "replaced by V2", StatusSuperseded, false},
		{"close a released plan", StatusReleased, PlanTypeBudget, ActionClose, "season complete", StatusClosed, false},
		{"a simulation is never submitted", StatusDraft, PlanTypeWhatIf, ActionSubmit, "", "", true},
		{"a simulation is never released", StatusApproved, PlanTypeWhatIf, ActionRelease, "", "", true},
		{"the actuals version is not approved", StatusInReview, PlanTypeActual, ActionApprove, "", "", true},
		{"the actuals version can be closed", StatusReleased, PlanTypeActual, ActionClose, "season complete", StatusClosed, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := PlanVersion{Status: c.status, PlanType: c.typ, Code: "V1"}
			tr, err := ApplyTransition(v, c.action, c.reason)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got a transition to %s", tr.To)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tr.To != c.wantTo {
				t.Errorf("transition to %s, want %s", tr.To, c.wantTo)
			}
			if tr.Permission == "" {
				t.Error("every transition must name the permission that guards it")
			}
		})
	}
}

func TestAllowedActions(t *testing.T) {
	got := AllowedActions(StatusDraft, PlanTypeBudget)
	if len(got) != 1 || got[0] != ActionSubmit {
		t.Errorf("draft actions = %v, want [SUBMIT]", got)
	}
	if got := AllowedActions(StatusDraft, PlanTypeWhatIf); len(got) != 0 {
		t.Errorf("a simulation offers no workflow actions, got %v", got)
	}
	if got := AllowedActions(StatusReleased, PlanTypeBudget); len(got) != 3 {
		t.Errorf("released actions = %v, want supersede, close and reopen", got)
	}
}

func TestDateLockingAndWritability(t *testing.T) {
	draft := PlanVersion{Status: StatusDraft, PlanType: PlanTypeBudget, Code: "V1"}
	if err := CheckWritable(draft, "2026-12-01", SeriesPlan); err != nil {
		t.Errorf("a draft must accept plan values: %v", err)
	}
	if err := CheckWritable(draft, "2026-12-01", SeriesActual); err == nil {
		t.Error("actuals must not be posted to a budget version")
	}

	released := PlanVersion{Status: StatusReleased, PlanType: PlanTypeBudget,
		Code: "V1", LockedThrough: "2026-12-31"}
	if err := CheckWritable(released, "2026-12-15", SeriesPlan); err == nil {
		t.Error("a date inside the locked period must be rejected")
	}
	if err := CheckWritable(released, "2027-01-15", SeriesPlan); err != nil {
		t.Errorf("a date beyond the lock date stays editable: %v", err)
	}

	actuals := PlanVersion{Status: StatusReleased, PlanType: PlanTypeActual, Code: "ACT"}
	if err := CheckWritable(actuals, "2026-12-15", SeriesActual); err != nil {
		t.Errorf("actuals must be postable while the season is open: %v", err)
	}
	if err := CheckWritable(actuals, "2026-12-15", SeriesPlan); err == nil {
		t.Error("plan values must not be written to the actuals version")
	}
	closed := PlanVersion{Status: StatusClosed, PlanType: PlanTypeActual, Code: "ACT"}
	if err := CheckWritable(closed, "2026-12-15", SeriesActual); err == nil {
		t.Error("a closed period must reject actual postings")
	}
}
