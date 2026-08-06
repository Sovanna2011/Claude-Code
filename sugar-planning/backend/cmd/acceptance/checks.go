package main

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/seed"
	"github.com/kss/sugarplan/internal/service"
)

// runAll walks section 27 in order. Each criterion is independent of the ones
// after it, and depends on the ones before it only through the ids the first
// criteria discover, so a failure early on shows up as a failure and not as
// nine more of them.
func runAll(r *run) {
	if err := r.discover(); err != nil {
		fmt.Printf("  %-4s %s\n       %v\n", fail, "discovering the instance", err)
		r.results = append(r.results, result{
			criterion: "C1", name: "discovering the instance", status: fail, detail: err.Error(),
		})
		return
	}

	for _, c := range []struct {
		code string
		f    func(*run)
	}{
		{"C1", planCreation},
		{"C2", approvalAndRelease},
		{"C3", operatorEntry},
		{"C4", derivedFigures},
		{"C5", postingsAtomicReversibleAuditable},
		{"C6", dashboardReconciles},
		{"C7", alertsAndThresholds},
		{"C8", roleAndTenantScope},
		{"C9", importErrorsBeforeCommit},
		{"C10", reportExports},
		{"C11", testsSecurityAndRestore},
	} {
		fmt.Printf("\n%d. %s\n", criteria[c.code].n, criteria[c.code].text)
		r.on(c.code, c.f)
	}
}

// ---------------------------------------------------------------------------
// Discovery
// ---------------------------------------------------------------------------

type page[T any] struct {
	Value []T `json:"value"`
	Count int `json:"count"`
}

type codedID struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// discover reads the master data the checks name things by. It uses codes
// rather than ids throughout, because an id is different on every instance and
// a code is the same one a person would use.
func (r *run) discover() error {
	var factories page[domain.Factory]
	if err := r.getOK("planner", "/api/v1/master/factories", &factories); err != nil {
		return err
	}
	for _, f := range factories.Value {
		if f.Code == seed.FactoryCode {
			r.state.factoryID, r.state.companyID = f.ID, f.CompanyID
		}
	}
	if r.state.factoryID == "" {
		return fmt.Errorf("factory %s is not in this instance; the harness expects the "+
			"reference scenario to be loaded (SEED_DEMO=true)", seed.FactoryCode)
	}

	r.state.products = map[string]string{}
	var products page[codedID]
	if err := r.getOK("planner", "/api/v1/master/products?$top=200", &products); err != nil {
		return err
	}
	for _, p := range products.Value {
		r.state.products[p.Code] = p.ID
	}
	r.state.productID = r.state.products["WHT"]

	r.state.warehouses = map[string]string{}
	var warehouses page[codedID]
	if err := r.getOK("planner", "/api/v1/master/warehouses?$top=200", &warehouses); err != nil {
		return err
	}
	for _, w := range warehouses.Value {
		r.state.warehouses[w.Code] = w.ID
	}
	r.state.warehouseID = r.state.warehouses["FG-WH1"]

	var packaging page[codedID]
	if err := r.getOK("planner", "/api/v1/master/packaging-types?$top=200", &packaging); err != nil {
		return err
	}
	for _, p := range packaging.Value {
		if p.Code == "P50KG" {
			r.state.packagingID = p.ID
		}
	}
	if r.state.productID == "" || r.state.warehouseID == "" || r.state.packagingID == "" {
		return fmt.Errorf("the reference master data is incomplete: product WHT, warehouse "+
			"FG-WH1 and packaging P50KG are needed and one of them is missing (%d products, "+
			"%d warehouses, %d packaging types)",
			len(products.Value), len(warehouses.Value), len(packaging.Value))
	}

	var seasons page[domain.Season]
	if err := r.getOK("planner", "/api/v1/seasons?$top=100", &seasons); err != nil {
		return err
	}
	for _, s := range seasons.Value {
		if s.Code == "2026-2027" {
			r.state.demoSeason = s.ID
		}
	}
	if r.state.demoSeason == "" {
		return fmt.Errorf("season 2026-2027 is not in this instance")
	}
	var versions page[domain.PlanVersion]
	if err := r.getOK("planner", "/api/v1/seasons/"+r.state.demoSeason+"/versions", &versions); err != nil {
		return err
	}
	for _, v := range versions.Value {
		if v.PlanType == domain.PlanTypeActual {
			r.state.demoActual = v.ID
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 1. A planner can create a season plan, enter assumptions, generate daily
//    targets, and compare versions.
// ---------------------------------------------------------------------------

// The harness builds its own small season rather than editing the reference
// one. Two reasons: the reference figures stay quotable after a run, and a
// thirty-day season is a plan whose arithmetic can be checked by hand.
const (
	accDays      = 30
	accCaneTons  = "300000"
	accRecovery  = "10.5"
	accStartDate = "2027-06-01"
)

func planCreation(r *run) {
	code := fmt.Sprintf("ACC-%d", time.Now().UTC().Unix())

	r.check("a planner creates a season", func() error {
		res, err := r.post("planner", "/api/v1/seasons", domain.Season{
			CompanyID: r.state.companyID, FactoryID: r.state.factoryID,
			Code: code, Name: "Acceptance harness season", StartDate: accStartDate,
			PlannedDays: accDays, Status: "OPEN",
		})
		if err != nil {
			return err
		}
		if res.Status != http.StatusCreated {
			return fmt.Errorf("expected 201, got %d %s", res.Status, res.snippet())
		}
		var season domain.Season
		if err := res.decode(&season); err != nil {
			return err
		}
		r.state.seasonID = season.ID
		return nil
	})
	if r.state.seasonID == "" {
		return // nothing after this can mean anything
	}

	r.check("a planner creates a draft version", func() error {
		res, err := r.post("planner", "/api/v1/seasons/"+r.state.seasonID+"/versions",
			domain.PlanVersion{
				Code: "V1", Description: "Acceptance baseline",
				PlanType: domain.PlanTypeBudget, Status: domain.StatusDraft,
				EffectiveFrom: accStartDate,
			})
		if err != nil {
			return err
		}
		if res.Status != http.StatusCreated {
			return fmt.Errorf("expected 201, got %d %s", res.Status, res.snippet())
		}
		var v domain.PlanVersion
		if err := res.decode(&v); err != nil {
			return err
		}
		r.state.versionID = v.ID
		return nil
	})
	if r.state.versionID == "" {
		return
	}

	r.check("assumptions are accepted and read back", func() error {
		for code, value := range accAssumptions() {
			res, err := r.put("planner", "/api/v1/versions/"+r.state.versionID+"/assumptions",
				domain.PlanAssumption{Code: code, Value: domain.D(value), UOM: assumptionUOM[code],
					Description: code})
			if err != nil {
				return err
			}
			if res.Status != http.StatusOK {
				return fmt.Errorf("saving %s: %d %s", code, res.Status, res.snippet())
			}
		}
		var v struct {
			Assumptions []domain.PlanAssumption `json:"assumptions"`
		}
		if err := r.getOK("planner", "/api/v1/versions/"+r.state.versionID, &v); err != nil {
			return err
		}
		got := map[string]domain.Dec{}
		for _, a := range v.Assumptions {
			got[a.Code] = a.Value
		}
		for code, want := range accAssumptions() {
			if !got[code].Equal(domain.D(want)) {
				return fmt.Errorf("%s reads back as %s, not %s", code, got[code], want)
			}
		}
		return nil
	})

	r.check("a product mix is accepted", func() error {
		res, err := r.put("planner", "/api/v1/versions/"+r.state.versionID+"/product-mix",
			domain.ProductMixEntry{
				ProductID: r.state.productID, PackagingID: r.state.packagingID,
				WarehouseID: r.state.warehouseID, SeasonTons: domain.D("20000"),
			})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("expected 200, got %d %s", res.Status, res.snippet())
		}
		return nil
	})

	r.check("generating the plan produces one row per crushing day", func() error {
		var gen service.GenerateResult
		res, err := r.post("planner", "/api/v1/versions/"+r.state.versionID+"/generate",
			service.GenerateRequest{Replace: true})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("expected 200, got %d %s", res.Status, res.snippet())
		}
		if err := res.decode(&gen); err != nil {
			return err
		}
		if gen.Summary.WorkingDays != accDays {
			return fmt.Errorf("the plan covers %d days, the season is %d",
				gen.Summary.WorkingDays, accDays)
		}
		// The whole target is allocated: a generator that quietly loses cane is
		// the failure this criterion is really about.
		if !gen.Summary.CaneAllocated.Equal(domain.D(accCaneTons)) {
			return fmt.Errorf("the plan allocates %s t of cane against a target of %s t",
				gen.Summary.CaneAllocated, accCaneTons)
		}
		return nil
	})

	r.check("the daily targets are readable and sum to the season target", func() error {
		var rows page[domain.DailyCanePlan]
		if err := r.getOK("planner",
			"/api/v1/versions/"+r.state.versionID+"/cane?$top=500", &rows); err != nil {
			return err
		}
		if len(rows.Value) != accDays {
			return fmt.Errorf("%d daily cane rows, want %d", len(rows.Value), accDays)
		}
		total := domain.Zero
		for _, row := range rows.Value {
			total = total.Add(row.CaneCrushed)
		}
		if !total.Equal(domain.D(accCaneTons)) {
			return fmt.Errorf("the daily rows sum to %s t, the season target is %s t",
				total, accCaneTons)
		}
		return nil
	})

	r.check("a what-if copy compares against the baseline", func() error {
		res, err := r.post("planner", "/api/v1/versions/"+r.state.versionID+"/copy",
			service.CopyRequest{
				Code: "V2", Description: "Acceptance what-if: lower recovery",
				PlanType: domain.PlanTypeForecast,
				AssumptionOverrides: map[string]domain.Dec{
					string(domain.AsmRecoveryPct): domain.D("9.5"),
				},
			})
		if err != nil {
			return err
		}
		if res.Status != http.StatusCreated {
			return fmt.Errorf("copying the version: %d %s", res.Status, res.snippet())
		}
		var v domain.PlanVersion
		if err := res.decode(&v); err != nil {
			return err
		}
		r.state.variantID = v.ID

		if res, err := r.post("planner", "/api/v1/versions/"+v.ID+"/generate",
			service.GenerateRequest{Replace: true}); err != nil {
			return err
		} else if res.Status != http.StatusOK {
			return fmt.Errorf("generating the what-if: %d %s", res.Status, res.snippet())
		}

		var cmp service.CompareResult
		res, err = r.post("planner", "/api/v1/versions/compare", service.CompareRequest{
			BaseVersionID: r.state.versionID, OtherVersionID: v.ID, Dimension: "DATE",
		})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("comparing: %d %s", res.Status, res.snippet())
		}
		if err := res.decode(&cmp); err != nil {
			return err
		}
		if len(cmp.Rows) == 0 {
			return fmt.Errorf("the comparison has no rows")
		}
		// The comparison has to say what was changed, not only that something
		// differs. A recovery of 9.5 against 10.5 is the one difference.
		found := false
		for _, d := range cmp.AssumptionDeltas {
			if strings.Contains(strings.ToUpper(d.Key), "RECOVERY") {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("the comparison does not report the changed recovery assumption; "+
				"deltas were %v", keysOf(cmp.AssumptionDeltas))
		}
		return nil
	})
}

func accAssumptions() map[string]string {
	return map[string]string{
		domain.AsmCaneTarget:        accCaneTons,
		domain.AsmSeasonDays:        fmt.Sprint(accDays),
		domain.AsmRecoveryPct:       accRecovery,
		domain.AsmDirectToRefinePct: "20",
		domain.AsmCrushRateTPH:      "500",
		domain.AsmAvailableHours:    "22",
		domain.AsmRemeltInputFactor: "1.05",
		domain.AsmQuotaShipmentTPD:  "300",
		domain.AsmCapacityWarnPct:   "85",
		domain.AsmCapacityAlertPct:  "95",
		domain.AsmMassBalanceTolPct: "0.5",
	}
}

var assumptionUOM = map[string]string{
	domain.AsmCaneTarget: "TON", domain.AsmSeasonDays: "DAY", domain.AsmRecoveryPct: "PCT",
	domain.AsmDirectToRefinePct: "PCT",
	domain.AsmCrushRateTPH:      "TPH", domain.AsmAvailableHours: "HOUR",
	domain.AsmRemeltInputFactor: "RATIO", domain.AsmQuotaShipmentTPD: "TPD",
	domain.AsmCapacityWarnPct: "PCT", domain.AsmCapacityAlertPct: "PCT",
	domain.AsmMassBalanceTolPct: "PCT",
}

func keysOf(rows []service.CompareRow) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Key)
	}
	return out
}

// ---------------------------------------------------------------------------
// 2. An approver can approve/reject and release a locked baseline with full
//    history.
// ---------------------------------------------------------------------------

func approvalAndRelease(r *run) {
	if r.state.versionID == "" {
		r.check("the workflow", func() error {
			return skipped("there is no version to move; criterion 1 did not get that far")
		})
		return
	}
	path := "/api/v1/versions/" + r.state.versionID + "/transition"

	r.check("a planner cannot approve their own plan", func() error {
		if res, err := r.post("planner", path,
			service.TransitionRequest{Action: domain.ActionSubmit}); err != nil {
			return err
		} else if res.Status != http.StatusOK {
			return fmt.Errorf("submitting: %d %s", res.Status, res.snippet())
		}
		res, err := r.post("planner", path,
			service.TransitionRequest{Action: domain.ActionApprove})
		if err != nil {
			return err
		}
		if res.Status != http.StatusForbidden {
			return fmt.Errorf("a planner approving their own plan was answered %d, want 403",
				res.Status)
		}
		return nil
	})

	r.check("a rejection needs a reason, and sends the plan back", func() error {
		res, err := r.post("approver", path, service.TransitionRequest{Action: domain.ActionReject})
		if err != nil {
			return err
		}
		if res.Status < 400 {
			return fmt.Errorf("a rejection with no reason was accepted with %d", res.Status)
		}
		res, err = r.post("approver", path, service.TransitionRequest{
			Action: domain.ActionReject, Reason: "the acceptance harness rejects it once",
		})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("rejecting: %d %s", res.Status, res.snippet())
		}
		var v domain.PlanVersion
		if err := res.decode(&v); err != nil {
			return err
		}
		if v.Status != domain.StatusRejected {
			return fmt.Errorf("the version reads %s after a rejection, want REJECTED", v.Status)
		}
		return nil
	})

	r.check("an approver approves and releases a baseline", func() error {
		for _, step := range []struct {
			user   string
			action domain.PlanAction
			want   domain.PlanStatus
		}{
			{"planner", domain.ActionSubmit, domain.StatusInReview},
			{"approver", domain.ActionApprove, domain.StatusApproved},
			{"approver", domain.ActionRelease, domain.StatusReleased},
		} {
			res, err := r.post(step.user, path, service.TransitionRequest{Action: step.action})
			if err != nil {
				return err
			}
			if res.Status != http.StatusOK {
				return fmt.Errorf("%s as %s: %d %s", step.action, step.user,
					res.Status, res.snippet())
			}
			var v domain.PlanVersion
			if err := res.decode(&v); err != nil {
				return err
			}
			if v.Status != step.want {
				return fmt.Errorf("after %s the version reads %s, want %s",
					step.action, v.Status, step.want)
			}
		}
		return nil
	})

	r.check("the released baseline is locked against further planning", func() error {
		res, err := r.post("planner", "/api/v1/versions/"+r.state.versionID+"/cane",
			map[string]any{"rows": []domain.DailyCanePlan{{
				VersionID: r.state.versionID, FactoryID: r.state.factoryID,
				BusinessDate: accStartDate, Series: domain.SeriesPlan,
				CaneCrushed: domain.D("1"), AvailableHrs: domain.D("22"),
			}}})
		if err != nil {
			return err
		}
		if res.Status < 400 {
			return fmt.Errorf("a released baseline accepted a planning write with %d; "+
				"a baseline that can still be edited is not a baseline", res.Status)
		}
		return nil
	})

	r.check("the whole history is in the audit trail", func() error {
		var events page[domain.AuditEvent]
		if err := r.getOK("auditor",
			"/api/v1/audit?entityId="+r.state.versionID+"&$top=200", &events); err != nil {
			return err
		}
		// Submit, reject, submit, approve, release. Every one of them has to be
		// there with the account that did it: a baseline whose history is
		// partial is a baseline nobody can defend in an audit.
		want := []string{"SUBMIT", "REJECT", "APPROVE", "RELEASE"}
		seen := map[string]string{}
		for _, e := range events.Value {
			for _, w := range want {
				if strings.Contains(strings.ToUpper(e.Action), w) {
					seen[w] = e.Actor
				}
			}
		}
		missing := []string{}
		for _, w := range want {
			if _, ok := seen[w]; !ok {
				missing = append(missing, w)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("the audit trail for this version is missing %s (it has %d events)",
				strings.Join(missing, ", "), len(events.Value))
		}
		if seen["RELEASE"] != "approver" {
			return fmt.Errorf("the release is recorded against %q, not the approver who made it",
				seen["RELEASE"])
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// 3. Operators can enter actual cane, production, packing, shipment, downtime
//    and quality data by shift/day.
// ---------------------------------------------------------------------------

func operatorEntry(r *run) {
	if r.state.demoActual == "" {
		r.check("operator entry", func() error {
			return skipped("the reference season has no actuals container to write to")
		})
		return
	}
	// A date past the fortnight of seeded actuals, so the harness writes where
	// nothing else does and the reference figures are left alone.
	day := domain.BusinessDate("2027-01-15")

	r.check("a supervisor records a day of crushing", func() error {
		res, err := r.post("supervisor", "/api/v1/versions/"+r.state.demoActual+"/cane",
			map[string]any{"rows": []domain.DailyCanePlan{{
				VersionID: r.state.demoActual, FactoryID: r.state.factoryID,
				BusinessDate: day, Series: domain.SeriesActual,
				CaneDelivered: domain.D("16000"), CaneAccepted: domain.D("15800"),
				CaneRejected: domain.D("200"), CaneCrushed: domain.D("15800"),
				CrushRateTPH: domain.D("718.182"), AvailableHrs: domain.D("24"),
				StoppageHrs: domain.D("2"), Note: "acceptance harness",
			}}})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("expected 200, got %d %s", res.Status, res.snippet())
		}
		var out service.UpsertResult
		if err := res.decode(&out); err != nil {
			return err
		}
		if out.Accepted != 1 {
			return fmt.Errorf("%d rows accepted, %d rejected: %v",
				out.Accepted, out.Rejected, out.Issues)
		}
		return nil
	})

	r.check("a supervisor records packed production", func() error {
		res, err := r.post("supervisor", "/api/v1/versions/"+r.state.demoActual+"/production",
			map[string]any{"rows": []domain.DailyProductPlan{{
				VersionID: r.state.demoActual, FactoryID: r.state.factoryID,
				BusinessDate: day, ProductID: r.state.productID,
				PackagingID: r.state.packagingID, Series: domain.SeriesActual,
				Quantity: domain.D("900"),
			}}})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("expected 200, got %d %s", res.Status, res.snippet())
		}
		return nil
	})

	r.check("a supervisor records a stoppage", func() error {
		res, err := r.post("supervisor", "/api/v1/downtime", domain.DowntimeEvent{
			FactoryID: r.state.factoryID, BusinessDate: day,
			StartAt:    mustTime(string(day) + "T02:00:00Z"),
			EndAt:      mustTime(string(day) + "T04:00:00Z"),
			ReasonCode: "DT-BOILER", RootCause: "acceptance harness",
		})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK && res.Status != http.StatusCreated {
			return fmt.Errorf("expected 200 or 201, got %d %s", res.Status, res.snippet())
		}
		var saved domain.DowntimeEvent
		if err := res.decode(&saved); err != nil {
			return err
		}
		// Two hours between the timestamps. The duration is the system's to
		// calculate; an operator typing it in is an operator who can get it
		// wrong.
		if !saved.DurationHrs.Equal(domain.D("2")) {
			return fmt.Errorf("a 02:00-04:00 stoppage was recorded as %s hours",
				saved.DurationHrs)
		}
		return nil
	})

	r.check("a shipment planner records a despatch", func() error {
		var channels page[codedID]
		if err := r.getOK("shipping", "/api/v1/master/shipment-channels", &channels); err != nil {
			return err
		}
		if len(channels.Value) == 0 {
			return fmt.Errorf("there are no shipment channels to despatch against")
		}
		res, err := r.post("shipping", "/api/v1/versions/"+r.state.demoActual+"/shipments",
			map[string]any{"rows": []domain.DailyShipmentPlan{{
				VersionID: r.state.demoActual, ChannelID: channels.Value[0].ID,
				ProductID: r.state.productID, WarehouseID: r.state.warehouseID,
				BusinessDate: day, Series: domain.SeriesActual, Quantity: domain.D("500"),
			}}})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("expected 200, got %d %s", res.Status, res.snippet())
		}
		return nil
	})

	r.check("the laboratory records a sample and its results", func() error {
		var params page[struct {
			ID   string `json:"id"`
			Code string `json:"code"`
		}]
		if err := r.getOK("quality", "/api/v1/quality/parameters", &params); err != nil {
			return err
		}
		if len(params.Value) == 0 {
			return fmt.Errorf("there are no quality parameters to record against")
		}
		res, err := r.post("quality", "/api/v1/quality/samples", service.SampleRequest{
			FactoryID: r.state.factoryID, BusinessDate: day,
			ProductID: r.state.productID, Comment: "acceptance harness",
		})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK && res.Status != http.StatusCreated {
			return fmt.Errorf("creating the sample: %d %s", res.Status, res.snippet())
		}
		var sample struct {
			ID string `json:"id"`
		}
		if err := res.decode(&sample); err != nil {
			return err
		}
		res, err = r.post("quality", "/api/v1/quality/samples/"+sample.ID+"/results",
			map[string]any{"complete": true, "results": []map[string]any{
				{"parameterId": params.Value[0].ID, "value": "99.800"},
			}})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("recording the results: %d %s", res.Status, res.snippet())
		}
		// The verdict is the system's: a laboratory that types "pass" into a
		// field has recorded an opinion, not a measurement against a limit.
		var graded service.ResultsOutcome
		if err := res.decode(&graded); err != nil {
			return err
		}
		if graded.Verdict == "" {
			return fmt.Errorf("the sheet came back without a verdict: %s", res.snippet())
		}
		if len(graded.Sample.Results) == 0 || graded.Sample.Results[0].Status == "" {
			return fmt.Errorf("the result came back ungraded: %s", res.snippet())
		}
		return nil
	})

	r.check("an operator cannot write outside their own job", func() error {
		// The other half of "operators can enter data": the ones who should not
		// be able to, cannot. A warehouse keeper has no business grading sugar.
		res, err := r.post("warehouse", "/api/v1/quality/samples", service.SampleRequest{
			FactoryID: r.state.factoryID, BusinessDate: day, ProductID: r.state.productID,
		})
		if err != nil {
			return err
		}
		if res.Status != http.StatusForbidden {
			return fmt.Errorf("a warehouse keeper creating a laboratory sample was "+
				"answered %d, want 403", res.Status)
		}
		return nil
	})
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// ---------------------------------------------------------------------------
// 4. The system automatically calculates cumulative totals, recovery,
//    balances, stock, capacity use, and forecast risk dates.
// ---------------------------------------------------------------------------

func derivedFigures(r *run) {
	var dash service.Dashboard
	loaded := r.check("the dashboard answers for the reference season", func() error {
		return r.getOK("executive", "/api/v1/dashboard?seasonId="+r.state.demoSeason, &dash)
	})
	if !loaded {
		return
	}

	r.check("cumulative totals are calculated, not stored", func() error {
		if dash.Cane.CumulativeActual.IsZero() {
			return fmt.Errorf("the cumulative actual is zero with a fortnight of crushing recorded")
		}
		// Cumulative plus remaining is the season target. It is the simplest
		// identity on the page and the one a wrong window silently breaks.
		sum := dash.Cane.CumulativeActual.Add(dash.Cane.Remaining)
		if !sum.Equal(dash.Cane.SeasonTarget) {
			return fmt.Errorf("cumulative %s + remaining %s = %s, but the season target is %s",
				dash.Cane.CumulativeActual, dash.Cane.Remaining, sum, dash.Cane.SeasonTarget)
		}
		return nil
	})

	r.check("recovery is derived from the two measured quantities", func() error {
		if dash.RawSugar.ActualTons.IsZero() || dash.Cane.CumulativeActual.IsZero() {
			return fmt.Errorf("there is no raw sugar or no cane to derive a recovery from")
		}
		want := domain.RoundPct(
			dash.RawSugar.ActualTons.Div(dash.Cane.CumulativeActual).Mul(domain.DI(100)))
		if !dash.RawSugar.ActualRecoveryPct.Equal(want) {
			return fmt.Errorf("the dashboard reports %s %% recovery; %s t of raw sugar on "+
				"%s t of cane is %s %%", dash.RawSugar.ActualRecoveryPct,
				dash.RawSugar.ActualTons, dash.Cane.CumulativeActual, want)
		}
		return nil
	})

	r.check("stock balances and capacity use are reported per warehouse", func() error {
		if len(dash.Storage) == 0 {
			return fmt.Errorf("no warehouse positions are reported")
		}
		for _, w := range dash.Storage {
			if w.CapacityTons.IsZero() {
				return fmt.Errorf("warehouse %s has no capacity, so its use cannot mean anything",
					w.WarehouseCode)
			}
			// Capacity use is balance over usable capacity, and it has to agree
			// with the two numbers printed beside it.
			want := domain.RoundPct(w.BalanceTons.Div(w.UsableTons).Mul(domain.DI(100)))
			if !w.CapacityUsePct.Equal(want) {
				return fmt.Errorf("%s reports %s %% used; %s t in %s t usable is %s %%",
					w.WarehouseCode, w.CapacityUsePct, w.BalanceTons, w.UsableTons, want)
			}
		}
		return nil
	})

	r.check("a forecast risk date is calculated for a store that fills", func() error {
		for _, w := range dash.Storage {
			if w.FirstFullDate != "" {
				if w.RequiredShipTPD.IsZero() {
					return fmt.Errorf("%s is forecast full on %s but no shipment rate is "+
						"given, which tells nobody what to do about it",
						w.WarehouseCode, w.FirstFullDate)
				}
				return nil
			}
		}
		return fmt.Errorf("no store is forecast to fill; the reference plan has two, so " +
			"either the figures or the forecast has changed")
	})

	r.check("the crushing forecast end date is calculated", func() error {
		if dash.Cane.ForecastEndDate == "" {
			return fmt.Errorf("no forecast end date is reported")
		}
		if dash.Cane.PlannedEndDate == "" {
			return fmt.Errorf("no planned end date is reported")
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// 5. Inventory and production postings are atomic, reversible through
//    documents, and auditable.
// ---------------------------------------------------------------------------

func postingsAtomicReversibleAuditable(r *run) {
	before, err := r.stock()
	if !r.check("the stock position is readable", func() error { return err }) {
		return
	}

	day := domain.BusinessDate("2027-01-16")
	var documentID, documentNo string

	r.check("a goods receipt moves the balance by exactly what was posted", func() error {
		res, err := r.post("warehouse", "/api/v1/inventory/documents", service.PostingRequest{
			DocType: domain.DocReceipt, BusinessDate: day, FactoryID: r.state.factoryID,
			Reference: "ACC-GR", Note: "acceptance harness",
			Lines: []service.PostingLineInput{{
				WarehouseID: r.state.warehouseID, ProductID: r.state.productID,
				Quantity: domain.D("120.500"), UOM: "TON",
			}},
		})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK && res.Status != http.StatusCreated {
			return fmt.Errorf("posting: %d %s", res.Status, res.snippet())
		}
		var doc domain.InventoryDocument
		if err := res.decode(&doc); err != nil {
			return err
		}
		documentID, documentNo = doc.ID, doc.DocumentNo

		after, err := r.stock()
		if err != nil {
			return err
		}
		moved := after[stockKey(r.state.warehouseID, r.state.productID)].
			Sub(before[stockKey(r.state.warehouseID, r.state.productID)])
		if !moved.Equal(domain.D("120.500")) {
			return fmt.Errorf("the balance moved by %s, the document posted 120.500", moved)
		}
		return nil
	})

	r.check("a document that fails half way leaves no half of it behind", func() error {
		snapshot, err := r.stock()
		if err != nil {
			return err
		}
		// Two lines: the first is good, the second names a product that does
		// not exist. Either both land or neither does.
		res, err := r.post("warehouse", "/api/v1/inventory/documents", service.PostingRequest{
			DocType: domain.DocReceipt, BusinessDate: day, FactoryID: r.state.factoryID,
			Reference: "ACC-ATOMIC",
			Lines: []service.PostingLineInput{
				{WarehouseID: r.state.warehouseID, ProductID: r.state.productID,
					Quantity: domain.D("10"), UOM: "TON"},
				{WarehouseID: r.state.warehouseID, ProductID: "00000000-0000-0000-0000-000000000000",
					Quantity: domain.D("10"), UOM: "TON"},
			},
		})
		if err != nil {
			return err
		}
		if res.Status < 400 {
			return fmt.Errorf("a document naming a product that does not exist was accepted "+
				"with %d", res.Status)
		}
		after, err := r.stock()
		if err != nil {
			return err
		}
		key := stockKey(r.state.warehouseID, r.state.productID)
		if !after[key].Equal(snapshot[key]) {
			return fmt.Errorf("the good line of a rejected document landed anyway: the "+
				"balance moved from %s to %s", snapshot[key], after[key])
		}
		return nil
	})

	r.check("the posting is reversible, and the reversal is its own document", func() error {
		if documentID == "" {
			return fmt.Errorf("there is no document to reverse")
		}
		snapshot, err := r.stock()
		if err != nil {
			return err
		}
		res, err := r.post("warehouse", "/api/v1/inventory/documents/"+documentID+"/reverse",
			map[string]string{"reason": "acceptance harness"})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK && res.Status != http.StatusCreated {
			return fmt.Errorf("reversing: %d %s", res.Status, res.snippet())
		}
		var reversal domain.InventoryDocument
		if err := res.decode(&reversal); err != nil {
			return err
		}
		if reversal.ID == documentID {
			return fmt.Errorf("the reversal is the same document; a corrected posting has " +
				"to leave both halves visible")
		}
		after, err := r.stock()
		if err != nil {
			return err
		}
		key := stockKey(r.state.warehouseID, r.state.productID)
		if !after[key].Equal(snapshot[key].Sub(domain.D("120.500"))) {
			return fmt.Errorf("after the reversal the balance is %s; it was %s and 120.500 "+
				"was taken back out", after[key], snapshot[key])
		}
		// And the original says it was reversed, so nobody reverses it twice.
		var original domain.InventoryDocument
		if err := r.getOK("warehouse", "/api/v1/inventory/documents/"+documentID, &original); err != nil {
			return err
		}
		if !original.Reversed {
			return fmt.Errorf("the original document does not say it was reversed")
		}
		if reversal.ReversalOf != documentID {
			return fmt.Errorf("the reversal does not point back at the document it undoes")
		}
		res, err = r.post("warehouse", "/api/v1/inventory/documents/"+documentID+"/reverse",
			map[string]string{"reason": "again"})
		if err != nil {
			return err
		}
		if res.Status < 400 {
			return fmt.Errorf("the same document was reversed twice, answered %d", res.Status)
		}
		return nil
	})

	r.check("both documents are in the audit trail with the account that posted them", func() error {
		if documentNo == "" {
			return fmt.Errorf("there is no document to look for")
		}
		var events page[domain.AuditEvent]
		if err := r.getOK("auditor", "/api/v1/audit?entity=inventory_document&$top=200",
			&events); err != nil {
			return err
		}
		for _, e := range events.Value {
			if e.EntityID == documentID {
				if e.Actor != "warehouse" {
					return fmt.Errorf("the posting is recorded against %q, not the keeper "+
						"who made it", e.Actor)
				}
				return nil
			}
		}
		return fmt.Errorf("document %s is not in the audit trail (%d stock events read)",
			documentNo, len(events.Value))
	})
}

func stockKey(warehouseID, productID string) string { return warehouseID + "/" + productID }

// stock reads the whole position as a map, so a check can say what moved rather
// than what the total is.
func (r *run) stock() (map[string]domain.Dec, error) {
	var lines page[service.StockLine]
	if err := r.getOK("warehouse", "/api/v1/stock?$top=500", &lines); err != nil {
		return nil, err
	}
	out := map[string]domain.Dec{}
	for _, l := range lines.Value {
		out[stockKey(l.WarehouseID, l.ProductID)] = l.Quantity
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 6. Dashboards reconcile to transaction data and the seeded scenario.
// ---------------------------------------------------------------------------

func dashboardReconciles(r *run) {
	var dash service.Dashboard
	loaded := r.check("the dashboard answers", func() error {
		return r.getOK("executive", "/api/v1/dashboard?seasonId="+r.state.demoSeason, &dash)
	})
	if !loaded {
		return
	}

	r.check("the downtime headline is the sum of the stoppage records", func() error {
		var events []domain.DowntimeEvent
		var body page[domain.DowntimeEvent]
		if err := r.getOK("supervisor", "/api/v1/downtime?$top=500", &body); err != nil {
			return err
		}
		events = body.Value
		hours := domain.Zero
		for _, e := range events {
			hours = hours.Add(e.DurationHrs)
		}
		if !dash.Downtime.Hours.Equal(hours) {
			return fmt.Errorf("the dashboard reports %s lost hours; the %d stoppage records "+
				"sum to %s", dash.Downtime.Hours, len(events), hours)
		}
		if dash.Downtime.EventCount != len(events) {
			return fmt.Errorf("the dashboard counts %d stoppages, there are %d records",
				dash.Downtime.EventCount, len(events))
		}
		return nil
	})

	r.check("the Pareto shares add up", func() error {
		if len(dash.Downtime.ByReason) == 0 {
			return fmt.Errorf("the downtime breakdown is empty")
		}
		last := dash.Downtime.ByReason[len(dash.Downtime.ByReason)-1]
		if !last.CumSharePct.Equal(domain.DI(100)) {
			return fmt.Errorf("the cumulative share ends at %s %%, want exactly 100",
				last.CumSharePct)
		}
		hours := domain.Zero
		for _, b := range dash.Downtime.ByReason {
			hours = hours.Add(b.Hours)
		}
		if !hours.Equal(dash.Downtime.Hours) {
			return fmt.Errorf("the reasons account for %s hours, the headline says %s",
				hours, dash.Downtime.Hours)
		}
		return nil
	})

	r.check("the crushing headline is the sum of the daily rows", func() error {
		var rows page[domain.DailyCanePlan]
		if err := r.getOK("planner", "/api/v1/versions/"+r.state.demoActual+
			"/cane?series=ACTUAL&$top=500", &rows); err != nil {
			return err
		}
		total := domain.Zero
		for _, row := range rows.Value {
			if row.BusinessDate <= dash.AsOf {
				total = total.Add(row.CaneCrushed)
			}
		}
		if !dash.Cane.CumulativeActual.Equal(total) {
			return fmt.Errorf("the dashboard reports %s t crushed to %s; the daily rows to "+
				"that date sum to %s t", dash.Cane.CumulativeActual, dash.AsOf, total)
		}
		return nil
	})

	r.check("the reference scenario's own figures are unchanged", func() error {
		// The numbers a demonstration is quoted on. If the harness or anything
		// else has moved them, that is worth failing over.
		if !dash.Cane.SeasonTarget.Equal(domain.D("2300000.000")) {
			return fmt.Errorf("the season target reads %s, the scenario is 2,300,000 t",
				dash.Cane.SeasonTarget)
		}
		total := domain.Zero
		for _, p := range dash.Products {
			total = total.Add(p.TargetTons)
		}
		if !total.Equal(domain.D("242100.000")) {
			return fmt.Errorf("the finished goods plan totals %s t, the scenario is 242,100 t",
				total)
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// 7. Capacity, shortage, quality, downtime and variance alerts work with
//    configurable thresholds.
// ---------------------------------------------------------------------------

func alertsAndThresholds(r *run) {
	r.check("evaluating the alerts fills the right inboxes", func() error {
		res, err := r.post("admin", "/api/v1/notifications/evaluate", map[string]string{
			"seasonId": r.state.demoSeason,
		})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("evaluating: %d %s", res.Status, res.snippet())
		}
		var inbox page[domain.Notification]
		if err := r.getOK("planner", "/api/v1/notifications?$top=100", &inbox); err != nil {
			return err
		}
		if len(inbox.Value) == 0 {
			return fmt.Errorf("the planner's inbox is empty; the reference plan has two " +
				"capacity problems in it")
		}
		return nil
	})

	r.check("an alert is addressed to a role, not to everybody", func() error {
		var planner, executive page[domain.Notification]
		if err := r.getOK("planner", "/api/v1/notifications?$top=100", &planner); err != nil {
			return err
		}
		if err := r.getOK("executive", "/api/v1/notifications?$top=100", &executive); err != nil {
			return err
		}
		for _, n := range planner.Value {
			if n.Recipient == "" {
				return fmt.Errorf("notification %s is addressed to nobody", n.Code)
			}
		}
		// The two inboxes must not be the same list, or "addressed to a role"
		// is a field nothing reads.
		if len(planner.Value) == len(executive.Value) && sameCodes(planner.Value, executive.Value) {
			return fmt.Errorf("the planner and the executive see the same %d notifications; "+
				"the recipient role is not being honoured", len(planner.Value))
		}
		return nil
	})

	r.check("the capacity threshold is read from the plan, not hard-coded", func() error {
		if r.state.variantID == "" {
			return skipped("there is no scenario version to change the threshold on")
		}
		// The what-if copy is still a draft, so its assumptions can be moved.
		// Warn at 1 % and the stores are in trouble from the first day; warn at
		// 99.9 % and they are not. A threshold that changes nothing is not a
		// threshold.
		read := func() (int, error) {
			var dash service.Dashboard
			if err := r.getOK("planner", "/api/v1/dashboard?seasonId="+r.state.seasonID+
				"&versionId="+r.state.variantID, &dash); err != nil {
				return 0, err
			}
			warned := 0
			for _, w := range dash.Storage {
				if w.Severity != domain.SeverityInfo && w.Severity != domain.SeveritySuccess {
					warned++
				}
			}
			return warned, nil
		}
		set := func(value string) error {
			res, err := r.put("planner", "/api/v1/versions/"+r.state.variantID+"/assumptions",
				domain.PlanAssumption{Code: domain.AsmCapacityWarnPct, Value: domain.D(value),
					UOM: "PCT", Description: "Capacity warning threshold"})
			if err != nil {
				return err
			}
			if res.Status != http.StatusOK {
				return fmt.Errorf("setting the threshold to %s: %d %s",
					value, res.Status, res.snippet())
			}
			return nil
		}

		if err := set("1"); err != nil {
			return err
		}
		strict, err := read()
		if err != nil {
			return err
		}
		if err := set("99.9"); err != nil {
			return err
		}
		lax, err := read()
		if err != nil {
			return err
		}
		if strict <= lax {
			return fmt.Errorf("warning at 1 %% flags %d stores and warning at 99.9 %% flags "+
				"%d; the threshold is not being read", strict, lax)
		}
		return nil
	})

	r.check("the plan generator warns about the shortages in the reference figures", func() error {
		var warnings []domain.Alert
		var v struct {
			Warnings []domain.Alert `json:"warnings"`
		}
		res, err := r.get("planner", "/api/v1/dashboard?seasonId="+r.state.demoSeason)
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("dashboard: %d %s", res.Status, res.snippet())
		}
		var dash service.Dashboard
		if err := res.decode(&dash); err != nil {
			return err
		}
		warnings = append(warnings, dash.Alerts...)
		warnings = append(warnings, v.Warnings...)
		if len(warnings) == 0 {
			return fmt.Errorf("the reference plan raises no alerts; it contains a raw sugar " +
				"shortfall and two stores that fill, and all three should be said out loud")
		}
		kinds := map[string]bool{}
		for _, a := range warnings {
			kinds[a.Code] = true
		}
		if len(kinds) < 2 {
			return fmt.Errorf("only one kind of alert is raised (%v); the criterion asks for "+
				"capacity, shortage, quality, downtime and variance", sortedKeys(kinds))
		}
		return nil
	})
}

func sameCodes(a, b []domain.Notification) bool {
	seen := map[string]int{}
	for _, n := range a {
		seen[n.Code+n.EntityID]++
	}
	for _, n := range b {
		seen[n.Code+n.EntityID]--
	}
	for _, v := range seen {
		if v != 0 {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// 8. Role and factory/company restrictions are enforced by the backend.
// ---------------------------------------------------------------------------

// This is the criterion the second tenant exists for. Every check here goes
// straight at the API with no browser involved, because a screen that hides a
// button is good manners and not a control.
func roleAndTenantScope(r *run) {
	r.check("a reader cannot write", func() error {
		res, err := r.post("executive", "/api/v1/seasons", domain.Season{
			CompanyID: r.state.companyID, FactoryID: r.state.factoryID,
			Code: "ACC-DENIED", Name: "should not exist", StartDate: accStartDate,
			PlannedDays: 1, Status: "OPEN",
		})
		if err != nil {
			return err
		}
		if res.Status != http.StatusForbidden {
			return fmt.Errorf("an executive viewer creating a season was answered %d, want 403",
				res.Status)
		}
		return nil
	})

	r.check("only the auditor can read the audit trail", func() error {
		if err := r.getOK("auditor", "/api/v1/audit?$top=1", nil); err != nil {
			return fmt.Errorf("the auditor cannot read the audit trail: %w", err)
		}
		for _, user := range []string{"planner", "warehouse", "executive"} {
			res, err := r.get(user, "/api/v1/audit?$top=1")
			if err != nil {
				return err
			}
			if res.Status != http.StatusForbidden {
				return fmt.Errorf("%s reading the audit trail was answered %d, want 403",
					user, res.Status)
			}
		}
		return nil
	})

	// Everything below needs the second tenant. Without it the only thing that
	// can be checked is that a caller scoped to nothing sees nothing, which is
	// not the same claim at all - so it is a skip and not a pass.
	var btb page[domain.Season]
	if err := r.getOK("btb-planner", "/api/v1/seasons?$top=100", &btb); err != nil {
		r.check("two tenants are kept apart", func() error {
			return skipped("this instance has no second tenant; boot it with SEED_TENANTS=true " +
				"(./test-system.sh does) to check data scope between two real factories")
		})
		return
	}
	for _, s := range btb.Value {
		if s.Code == seed.SecondSeasonCode {
			r.state.btbSeason = s.ID
		}
	}

	r.check("each tenant's season list holds only its own seasons", func() error {
		if r.state.btbSeason == "" {
			return fmt.Errorf("the second tenant cannot see its own season %s",
				seed.SecondSeasonCode)
		}
		for _, s := range btb.Value {
			if s.FactoryID == r.state.factoryID {
				return fmt.Errorf("the second tenant's planner can see season %s, which "+
					"belongs to %s", s.Code, seed.FactoryCode)
			}
		}
		var kss page[domain.Season]
		if err := r.getOK("planner", "/api/v1/seasons?$top=100", &kss); err != nil {
			return err
		}
		for _, s := range kss.Value {
			if s.Code == seed.SecondSeasonCode {
				return fmt.Errorf("the first tenant's planner can see the second tenant's season")
			}
		}
		return nil
	})

	r.check("naming another tenant's record by id does not get past the scope", func() error {
		for _, c := range []struct {
			user, path, what string
		}{
			{"btb-planner", "/api/v1/seasons/" + r.state.demoSeason, "the first tenant's season"},
			{"btb-planner", "/api/v1/dashboard?seasonId=" + r.state.demoSeason,
				"the first tenant's dashboard"},
			{"planner", "/api/v1/seasons/" + r.state.btbSeason, "the second tenant's season"},
			{"planner", "/api/v1/dashboard?seasonId=" + r.state.btbSeason,
				"the second tenant's dashboard"},
		} {
			res, err := r.get(c.user, c.path)
			if err != nil {
				return err
			}
			if res.Status != http.StatusNotFound && res.Status != http.StatusForbidden {
				return fmt.Errorf("%s reading %s by id was answered %d, want 404 or 403",
					c.user, c.what, res.Status)
			}
		}
		return nil
	})

	r.check("stock cannot be posted into another tenant's warehouse", func() error {
		res, err := r.post("btb-warehouse", "/api/v1/inventory/documents", service.PostingRequest{
			DocType: domain.DocReceipt, BusinessDate: "2027-01-17",
			FactoryID: r.state.factoryID, Reference: "ACC-CROSS",
			Lines: []service.PostingLineInput{{
				WarehouseID: r.state.warehouseID, ProductID: r.state.productID,
				Quantity: domain.D("1"), UOM: "TON",
			}},
		})
		if err != nil {
			return err
		}
		if res.Status < 400 {
			return fmt.Errorf("the second tenant's keeper posted stock into %s's warehouse "+
				"and was answered %d", seed.FactoryCode, res.Status)
		}
		return nil
	})

	r.check("each tenant's stock position holds only its own warehouses", func() error {
		var theirs page[service.StockLine]
		if err := r.getOK("btb-warehouse", "/api/v1/stock?$top=500", &theirs); err != nil {
			return err
		}
		for _, l := range theirs.Value {
			if l.WarehouseID == r.state.warehouseID {
				return fmt.Errorf("the second tenant's keeper can see stock in %s",
					l.WarehouseCode)
			}
		}
		return nil
	})

	r.check("the two tenants' plans are different plans", func() error {
		// The second tenant is seeded with figures of its own precisely so that
		// a leak cannot pass as a pass: 640,000 t against 2,300,000 t.
		var dash service.Dashboard
		if err := r.getOK("btb-planner", "/api/v1/dashboard?seasonId="+r.state.btbSeason,
			&dash); err != nil {
			return err
		}
		if !dash.Cane.SeasonTarget.Equal(domain.D("640000.000")) {
			return fmt.Errorf("the second tenant's season target reads %s, want 640,000 t",
				dash.Cane.SeasonTarget)
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// 9. Excel import identifies errors before commit and preserves source-row
//    traceability.
// ---------------------------------------------------------------------------

func importErrorsBeforeCommit(r *run) {
	if r.state.demoActual == "" {
		r.check("import", func() error {
			return skipped("there is no actuals container to import into")
		})
		return
	}

	// Four rows against the CANE-DAILY template. Rows 2 and 4 are good; row 3
	// carries a tonnage nobody can parse and row 5 a date nobody can. The row
	// numbers are the file's own, counting the header as row 1, which is what
	// "source-row traceability" has to mean to be any use at the desk.
	file := strings.Join([]string{
		"Date,Delivered (MT),Accepted (MT),Rejected (MT),Crushed (MT),Rate (TPH),Hours available,Hours stopped,Remarks",
		"20/01/2027,15000,14900,100,14900,700,24,0,good row",
		"21/01/2027,15000,14900,100,not a number,700,24,0,bad tonnage",
		"22/01/2027,15000,14900,100,14900,700,24,0,good row",
		"31/02/2027,15000,14900,100,14900,700,24,0,no such date",
		"",
	}, "\n")

	var jobID string
	staged := r.check("a file with bad rows in it stages without committing anything", func() error {
		path := fmt.Sprintf("/api/v1/imports?mapping=CANE-DAILY&versionId=%s&series=ACTUAL&fileName=acceptance.csv",
			r.state.demoActual)
		res, err := r.do("supervisor", http.MethodPost, path, []byte(file), nil)
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK && res.Status != http.StatusCreated {
			return fmt.Errorf("staging: %d %s", res.Status, res.snippet())
		}
		// The response is the job and the staged rows together, which is what
		// the preview screen renders: the file, as the system read it, before
		// anybody commits it.
		var staged struct {
			Job  domain.ImportJob   `json:"job"`
			Rows []domain.ImportRow `json:"rows"`
		}
		if err := res.decode(&staged); err != nil {
			return err
		}
		job := staged.Job
		jobID = job.ID
		if len(staged.Rows) != 4 {
			return fmt.Errorf("the file has four rows and %d were staged", len(staged.Rows))
		}
		if job.ErrorRows == 0 {
			return fmt.Errorf("the file has two unusable rows in it and staging found none")
		}
		if job.Status == domain.ImportCommitted {
			return fmt.Errorf("staging committed the file; the errors are supposed to be " +
				"found before anything lands")
		}
		return nil
	})
	if !staged {
		return
	}

	r.check("every error names the row it came from and what was wrong", func() error {
		// The error list downloads as a CSV, because what somebody does with it
		// is open it beside the file they uploaded. Row, Field, Problem, Value.
		res, err := r.get("supervisor", "/api/v1/imports/"+jobID+"/errors")
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("%d %s", res.Status, res.snippet())
		}
		records, err := csv.NewReader(strings.NewReader(string(res.Body))).ReadAll()
		if err != nil {
			return fmt.Errorf("the error list is not readable as CSV: %w", err)
		}
		if len(records) < 3 { // the header plus one line per bad row
			return fmt.Errorf("%d error lines reported, the file has two unusable rows",
				len(records)-1)
		}
		rows := map[int]bool{}
		for _, line := range records[1:] {
			if len(line) < 4 {
				return fmt.Errorf("an error line is missing columns: %v", line)
			}
			n, err := strconv.Atoi(line[0])
			if err != nil || n == 0 {
				return fmt.Errorf("an error carries no source row: %v", line)
			}
			if strings.TrimSpace(line[2]) == "" {
				return fmt.Errorf("row %d is marked bad with nothing said about why", n)
			}
			rows[n] = true
		}
		// Rows 3 and 5 of the file, counting the header. Not "two errors
		// somewhere": the point of the criterion is that somebody can open the
		// spreadsheet and go straight to the cell.
		for _, want := range []int{3, 5} {
			if !rows[want] {
				return fmt.Errorf("the errors are on rows %v; rows 3 and 5 are the bad ones",
					sortedInts(rows))
			}
		}
		return nil
	})

	r.check("committing is refused while the file still has errors", func() error {
		res, err := r.post("supervisor", "/api/v1/imports/"+jobID+"/commit", nil)
		if err != nil {
			return err
		}
		if res.Status < 400 {
			return fmt.Errorf("a file with two unusable rows committed with %d %s",
				res.Status, res.snippet())
		}
		return nil
	})

	r.check("the staged file is cancellable, and nothing of it was written", func() error {
		res, err := r.post("supervisor", "/api/v1/imports/"+jobID+"/cancel",
			map[string]string{"reason": "acceptance harness"})
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("cancelling: %d %s", res.Status, res.snippet())
		}
		// The good rows of a cancelled file must not be in the plan.
		var rows page[domain.DailyCanePlan]
		if err := r.getOK("planner", "/api/v1/versions/"+r.state.demoActual+
			"/cane?from=2027-01-20&to=2027-01-22&$top=50", &rows); err != nil {
			return err
		}
		if len(rows.Value) > 0 {
			return fmt.Errorf("%d rows from the cancelled file are in the plan", len(rows.Value))
		}
		return nil
	})
}

func sortedInts(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// ---------------------------------------------------------------------------
// 10. Reports export correctly to Excel, PDF and CSV.
// ---------------------------------------------------------------------------

func reportExports(r *run) {
	var catalogue page[struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}]
	listed := r.check("the report catalogue is readable", func() error {
		if err := r.getOK("executive", "/api/v1/reports", &catalogue); err != nil {
			return err
		}
		if len(catalogue.Value) == 0 {
			return fmt.Errorf("the catalogue is empty")
		}
		return nil
	})
	if !listed {
		return
	}

	query := "?seasonId=" + r.state.demoSeason
	for _, format := range []struct {
		name, ext, contentType string
		magic                  []byte
	}{
		{"CSV", "csv", "text/csv", nil},
		{"Excel", "xlsx", "spreadsheetml", []byte("PK")},
		{"PDF", "pdf", "application/pdf", []byte("%PDF")},
	} {
		r.check(fmt.Sprintf("every report exports as %s", format.name), func() error {
			var broken []string
			for _, def := range catalogue.Value {
				res, err := r.get("executive",
					"/api/v1/reports/"+def.Code+query+"&format="+format.ext)
				if err != nil {
					return err
				}
				switch {
				case res.Status != http.StatusOK:
					broken = append(broken, fmt.Sprintf("%s: %d %s",
						def.Code, res.Status, res.snippet()))
				case len(res.Body) == 0:
					broken = append(broken, def.Code+": empty file")
				case !strings.Contains(res.Header.Get("Content-Type"), format.contentType):
					broken = append(broken, fmt.Sprintf("%s: content type %q",
						def.Code, res.Header.Get("Content-Type")))
				case format.magic != nil && !hasPrefix(res.Body, format.magic):
					broken = append(broken, fmt.Sprintf("%s: the body is not %s",
						def.Code, format.name))
				case !strings.Contains(res.Header.Get("Content-Disposition"), "."+format.ext):
					broken = append(broken, fmt.Sprintf("%s: no file name in the response",
						def.Code))
				}
			}
			if len(broken) > 0 {
				return fmt.Errorf("%d of %d reports: %s", len(broken), len(catalogue.Value),
					strings.Join(broken, "; "))
			}
			return nil
		})
	}

	r.check("an export carries the header that makes it quotable", func() error {
		res, err := r.get("executive", "/api/v1/reports/"+catalogue.Value[0].Code+query+"&format=csv")
		if err != nil {
			return err
		}
		if res.Status != http.StatusOK {
			return fmt.Errorf("%d %s", res.Status, res.snippet())
		}
		// A report printed off and carried into a meeting has to say what it is
		// of and when it was taken, or it is a page of numbers.
		body := string(res.Body)
		for _, want := range []string{"Kampong Speu", "2026-2027"} {
			if !strings.Contains(body, want) {
				return fmt.Errorf("the export does not name %q anywhere in its header", want)
			}
		}
		return nil
	})

	// This check exists because the first run of this harness found the
	// opposite: the packaging requirement report needs materials:read, which
	// only the planner holds, and it was on everybody's catalogue answering 403.
	r.check("the catalogue offers nothing it will then refuse", func() error {
		for _, user := range []string{"executive", "warehouse", "quality", "auditor"} {
			var theirs page[struct {
				Code string `json:"code"`
			}]
			if err := r.getOK(user, "/api/v1/reports", &theirs); err != nil {
				return err
			}
			for _, def := range theirs.Value {
				res, err := r.get(user, "/api/v1/reports/"+def.Code+query+"&format=csv")
				if err != nil {
					return err
				}
				if res.Status == http.StatusForbidden {
					return fmt.Errorf("%s is offered %s and refused it: %s",
						user, def.Code, res.snippet())
				}
			}
		}
		// And the other half: a report left off the catalogue is refused when
		// named directly, because hiding it is a courtesy and not a control.
		res, err := r.get("executive", "/api/v1/reports/material-requirements"+query+"&format=csv")
		if err != nil {
			return err
		}
		if res.Status != http.StatusForbidden {
			return fmt.Errorf("an executive viewer naming the packaging requirement report "+
				"directly was answered %d, want 403", res.Status)
		}
		return nil
	})
}

func hasPrefix(body, prefix []byte) bool {
	if len(body) < len(prefix) {
		return false
	}
	for i := range prefix {
		if body[i] != prefix[i] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// 11. Automated tests pass, no critical security findings remain, and
//     backup/restore has been demonstrated.
// ---------------------------------------------------------------------------

func testsSecurityAndRestore(r *run) {
	// Three of these are not questions an HTTP client can answer, and pretending
	// otherwise would be the worst thing a harness could do. They are checked by
	// test-system.sh, which runs the suites and the restore drill around this
	// program and fails the run if any of them fails.
	r.check("the Go and JavaScript test suites pass", func() error {
		return skipped("run by test-system.sh: go test ./... and npm test")
	})
	r.check("no critical security findings remain", func() error {
		return skipped("run by test-system.sh: govulncheck ./... and go vet ./...")
	})
	r.check("backup and restore has been demonstrated", func() error {
		return skipped("run by test-system.sh --postgres: pg_dump, restore into a scratch " +
			"database, and a row count comparison; the drill is in docs/runbook.md §3")
	})

	// What this program *can* check is that the running instance says who it is
	// and is fit to be scraped and watched, which is the part of the criterion
	// that lives at the other end of a URL.
	r.check("the instance reports its health, readiness and metrics", func() error {
		for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
			res, err := r.get("", path)
			if err != nil {
				return err
			}
			if res.Status != http.StatusOK {
				return fmt.Errorf("%s answered %d", path, res.Status)
			}
		}
		res, err := r.get("", "/metrics")
		if err != nil {
			return err
		}
		if !strings.Contains(string(res.Body), "http_requests_total") {
			return fmt.Errorf("/metrics carries no request counter")
		}
		return nil
	})

	r.check("an unauthenticated caller gets nothing", func() error {
		for _, path := range []string{
			"/api/v1/seasons", "/api/v1/dashboard?seasonId=" + r.state.demoSeason,
			"/api/v1/stock", "/api/v1/audit",
		} {
			res, err := r.get("", path)
			if err != nil {
				return err
			}
			if res.Status != http.StatusUnauthorized {
				return fmt.Errorf("%s answered an anonymous caller %d, want 401", path, res.Status)
			}
		}
		return nil
	})

	r.check("a refused request explains itself in the documented format", func() error {
		res, err := r.get("planner", "/api/v1/audit?$top=1")
		if err != nil {
			return err
		}
		if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "application/problem+json") {
			return fmt.Errorf("a refusal came back as %q, not application/problem+json", ct)
		}
		var problem struct {
			Type   string `json:"type"`
			Title  string `json:"title"`
			Status int    `json:"status"`
		}
		if err := res.decode(&problem); err != nil {
			return err
		}
		if problem.Title == "" || problem.Status != http.StatusForbidden {
			return fmt.Errorf("the problem document is incomplete: %s", res.snippet())
		}
		return nil
	})
}
