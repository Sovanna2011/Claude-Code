package service

import (
	"context"
	"fmt"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// GenerateRequest asks for the daily plan to be rebuilt from the version's
// assumptions and product mix.
type GenerateRequest struct {
	// NonWorkingDays are extra dates excluded from crushing, on top of the
	// approved maintenance windows the generator picks up on its own. The
	// campaign is extended rather than shortened.
	NonWorkingDays []domain.BusinessDate `json:"nonWorkingDays,omitempty"`
	// Replace clears the version's existing daily rows first. Without it, a
	// generate run refuses to overwrite work that is already there.
	Replace bool `json:"replace"`
}

// GenerateResult reports what the generator produced.
type GenerateResult struct {
	Summary   domain.PlanSummary  `json:"summary"`
	Warnings  []domain.Alert      `json:"warnings"`
	RowCounts map[string]int      `json:"rowCounts"`
	FirstDate domain.BusinessDate `json:"firstDate"`
	LastDate  domain.BusinessDate `json:"lastDate"`
	// NonWorkingDays are the dates the run skipped. The planner needs to see
	// them: they are the reason the campaign ends later than the day count
	// alone suggests.
	NonWorkingDays []domain.BusinessDate `json:"nonWorkingDays,omitempty"`
}

// Generate rebuilds the daily plan for a version.
//
// Everything happens in one transaction: either the version ends up with a
// complete, consistent plan, or it is left exactly as it was.
func (p *Planning) Generate(ctx context.Context, versionID string, req GenerateRequest) (GenerateResult, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanWrite); err != nil {
		return GenerateResult{}, err
	}
	version, err := p.versionInScope(ctx, versionID)
	if err != nil {
		return GenerateResult{}, err
	}
	if !version.IsEditable() {
		return GenerateResult{}, fmt.Errorf(
			"%w: version %s is %s; copy it into a new scenario to change the plan",
			domain.ErrLocked, version.Code, version.Status)
	}
	if version.PlanType == domain.PlanTypeActual {
		return GenerateResult{}, fmt.Errorf(
			"%w: the actuals container records what happened and is not generated", domain.ErrValidation)
	}

	season, err := p.store.Planning().GetSeason(ctx, version.SeasonID)
	if err != nil {
		return GenerateResult{}, err
	}

	input, err := p.buildGeneratorInput(ctx, season, version, req)
	if err != nil {
		return GenerateResult{}, err
	}

	output, err := domain.Generate(*input)
	if err != nil {
		return GenerateResult{}, err
	}

	result := GenerateResult{
		Summary:  output.Summary,
		Warnings: output.Warnings,
		RowCounts: map[string]int{
			"cane": len(output.Cane), "production": len(output.Products),
			"storage": len(output.Storage), "shipment": len(output.Shipments),
		},
	}
	if len(output.Dates) > 0 {
		result.FirstDate, result.LastDate = output.Dates[0], output.Dates[len(output.Dates)-1]
		// Only the skipped days that fall inside the campaign are reported; a
		// maintenance window booked for next season is not this plan's news.
		for _, d := range sortedDates(input.NonWorkingDays) {
			if d >= result.FirstDate && d <= result.LastDate {
				result.NonWorkingDays = append(result.NonWorkingDays, d)
			}
		}
	}

	err = p.store.InTx(ctx, func(tx store.Store) error {
		existing, err := tx.Planning().ListCane(ctx, store.PlanFilter{VersionIDs: []string{versionID}, Top: 1})
		if err != nil {
			return err
		}
		if len(existing) > 0 && !req.Replace {
			return fmt.Errorf(
				"%w: version %s already has a daily plan; send replace=true to rebuild it",
				domain.ErrValidation, version.Code)
		}
		if err := tx.Planning().DeleteVersionRows(ctx, versionID); err != nil {
			return err
		}
		if _, err := tx.Planning().UpsertCane(ctx, output.Cane, caller.Username); err != nil {
			return err
		}
		if _, err := tx.Planning().UpsertProducts(ctx, output.Products, caller.Username); err != nil {
			return err
		}
		if err := p.writeStorageWithBalances(ctx, tx, input, output.Storage, caller.Username); err != nil {
			return err
		}
		if _, err := tx.Planning().UpsertShipments(ctx, output.Shipments, caller.Username); err != nil {
			return err
		}

		// The season end date follows the generated campaign, so the header and
		// the daily rows cannot disagree.
		if result.LastDate != "" && season.EndDate != result.LastDate {
			season.EndDate = result.LastDate
			if _, err := tx.Planning().SaveSeason(ctx, season, caller.Username); err != nil {
				return err
			}
		}
		return p.audit(ctx, tx, auditEntry{
			action: "GENERATE", entity: "plan_version", entityID: versionID,
			after:  result.Summary,
			reason: fmt.Sprintf("generated %d days", len(output.Dates)),
		})
	})
	if err != nil {
		return GenerateResult{}, err
	}
	return result, nil
}

// writeStorageWithBalances rolls the generated movements forward through the
// ledger so that every stored row carries its beginning and ending balance.
// Storing the balances rather than deriving them on read is what lets the
// planning board and the reports show a stock position without replaying the
// whole season for every request.
func (p *Planning) writeStorageWithBalances(ctx context.Context, tx store.Store,
	in *domain.GeneratorInput, rows []domain.DailyStoragePlan, actor string) error {

	byStore := map[string][]domain.DailyStoragePlan{}
	for _, r := range rows {
		key := r.WarehouseID + "|" + r.ProductID
		byStore[key] = append(byStore[key], r)
	}

	var out []domain.DailyStoragePlan
	for key, group := range byStore {
		sortStorage(group)
		movements := make([]domain.LedgerMovement, len(group))
		for i, r := range group {
			movements[i] = domain.LedgerMovement{
				Date: r.BusinessDate, ProductionReceipt: r.ProductionReceipt,
				TransferIn: r.TransferIn, RepackIn: r.RepackIn, Adjustment: r.Adjustment,
				RemeltIssue: r.RemeltIssue, ShipmentQty: r.ShipmentQty,
				TransferOut: r.TransferOut, RepackOut: r.RepackOut, ProcessLoss: r.ProcessLoss,
				HoldQty: r.HoldQty,
			}
		}
		warehouseID := key[:len(key)-len(group[0].ProductID)-1]
		wh := in.Warehouses[warehouseID]
		days := domain.RollLedger(wh.OpeningBalance, wh.UsableCapacity(), movements)
		for i := range group {
			group[i].BeginningBalance = days[i].BeginningBalance
			group[i].EndingBalance = days[i].EndingBalance
		}
		out = append(out, group...)
	}
	_, err := tx.Planning().UpsertStorage(ctx, out, actor)
	return err
}

func sortStorage(rows []domain.DailyStoragePlan) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].BusinessDate < rows[j-1].BusinessDate; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}

// buildGeneratorInput gathers the master data and assumptions the generator
// needs, and reports precisely what is missing rather than failing obscurely.
func (p *Planning) buildGeneratorInput(ctx context.Context, season domain.Season,
	version domain.PlanVersion, req GenerateRequest) (*domain.GeneratorInput, error) {

	pl := p.store.Planning()
	md := p.store.MasterData()

	assumptionRows, err := pl.ListAssumptions(ctx, version.ID)
	if err != nil {
		return nil, err
	}
	assumptions := map[string]domain.Dec{}
	for _, a := range assumptionRows {
		assumptions[a.Code] = a.Value
	}
	if _, ok := assumptions[domain.AsmSeasonDays]; !ok && season.PlannedDays > 0 {
		assumptions[domain.AsmSeasonDays] = domain.DI(int64(season.PlannedDays))
	}

	mix, err := pl.ListMix(ctx, version.ID)
	if err != nil {
		return nil, err
	}

	active := true
	productPage, err := md.Products().List(ctx, store.ListOptions{Top: 1000, Active: &active})
	if err != nil {
		return nil, err
	}
	products := map[string]domain.Product{}
	rawProductID := ""
	for _, pr := range productPage.Items {
		products[pr.ID] = pr
		// The raw sugar stock product is the active raw-class product with the
		// lowest code; sites with more than one grade of raw sugar set the
		// warehouse assignment explicitly in the product mix instead.
		if pr.StorageClass == domain.StorageRaw && (rawProductID == "" || pr.Code < products[rawProductID].Code) {
			rawProductID = pr.ID
		}
	}

	warehousePage, err := md.Warehouses().List(ctx, store.ListOptions{
		Top: 1000, Active: &active, ParentID: season.FactoryID,
	})
	if err != nil {
		return nil, err
	}
	warehouses := map[string]domain.Warehouse{}
	var rawWarehouses []string
	for _, w := range warehousePage.Items {
		warehouses[w.ID] = w
		if w.StorageClass == domain.StorageRaw {
			rawWarehouses = append(rawWarehouses, w.ID)
		}
	}

	channelPage, err := md.Channels().List(ctx, store.ListOptions{Top: 1000, Active: &active})
	if err != nil {
		return nil, err
	}
	quotaChannel := ""
	for _, c := range channelPage.Items {
		if c.Category == "QUOTA" && (quotaChannel == "" || c.Code < quotaChannel) {
			quotaChannel = c.ID
			break
		}
	}

	verr := &domain.ValidationError{}
	if len(mix) == 0 {
		verr.Add("productMix", "EMPTY",
			"the version has no product mix; add the finished goods tonnage before generating")
	}
	if rawProductID == "" {
		verr.Add("masterData.products", "MISSING_RAW_PRODUCT",
			"no active product with storage class RAW was found to hold raw sugar stock")
	}
	if len(rawWarehouses) == 0 {
		verr.Add("masterData.warehouses", "MISSING_RAW_WAREHOUSE",
			"no active raw sugar warehouse was found for this factory")
	}
	if quotaChannel == "" {
		verr.Add("masterData.channels", "MISSING_QUOTA_CHANNEL",
			"no active shipment channel of category QUOTA was found to carry the shipment plan")
	}
	if err := verr.OrNil(); err != nil {
		return nil, err
	}

	// Approved maintenance removes crushing days without anybody having to
	// remember to type them into the request. Only approved, factory-wide
	// windows count: a window somebody is still thinking about must not quietly
	// move the end of the season, and a line outage does not stop the mill.
	windows, err := p.store.Execution().ListMaintenance(ctx, store.ExecutionFilter{
		FactoryID: season.FactoryID,
		From:      season.StartDate,
		To:        season.EndDate,
		Statuses:  []string{domain.MaintenanceApproved},
	})
	if err != nil {
		return nil, err
	}
	nonWorking := domain.NonWorkingDays(windows)

	// Anything the caller sends is added on top, for a shutdown that has not
	// been entered as a maintenance window.
	for _, d := range req.NonWorkingDays {
		if !d.Valid() {
			return nil, fmt.Errorf("%w: %q is not a valid non-working date", domain.ErrValidation, d)
		}
		nonWorking[d] = true
	}

	return &domain.GeneratorInput{
		Season: season, Version: version, Assumptions: assumptions, Mix: mix,
		Products: products, Warehouses: warehouses, RawWarehouseIDs: rawWarehouses,
		RawProductID: rawProductID, QuotaChannelID: quotaChannel, NonWorkingDays: nonWorking,
	}, nil
}
