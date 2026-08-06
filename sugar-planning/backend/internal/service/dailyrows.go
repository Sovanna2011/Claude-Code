package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// RowIssue reports a problem with one row of a bulk request, addressed by its
// position in the payload so the SAPUI5 table can highlight the right line.
type RowIssue struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// UpsertResult is the outcome of a bulk write.
type UpsertResult struct {
	Accepted int        `json:"accepted"`
	Rejected int        `json:"rejected"`
	Issues   []RowIssue `json:"issues,omitempty"`
}

// UpsertOptions controls how strictly a bulk write is applied.
type UpsertOptions struct {
	// Partial accepts the valid rows and reports the rest. The default is
	// all-or-nothing, which is what a planning grid save wants: the planner
	// sees one error and fixes it, rather than discovering later that half the
	// week saved.
	Partial bool
}

// ---------------------------------------------------------------------------
// Cane
// ---------------------------------------------------------------------------

// UpsertCane writes daily cane rows, planned or actual.
func (p *Planning) UpsertCane(ctx context.Context, versionID string, rows []domain.DailyCanePlan,
	opts UpsertOptions) (UpsertResult, error) {

	caller := auth.FromContext(ctx)
	version, season, err := p.writeTarget(ctx, versionID)
	if err != nil {
		return UpsertResult{}, err
	}

	var issues []RowIssue
	valid := make([]domain.DailyCanePlan, 0, len(rows))
	for i, r := range rows {
		r.VersionID = versionID
		if r.FactoryID == "" {
			r.FactoryID = season.FactoryID
		}
		if bad := p.checkRow(caller, version, i, r.BusinessDate, r.Series, permForSeries(r.Series, domain.PermActualCane)); bad != nil {
			issues = append(issues, *bad)
			continue
		}
		if err := caller.RequireFactory(r.FactoryID); err != nil {
			issues = append(issues, RowIssue{Row: i, Field: "factoryId", Code: "FORBIDDEN", Message: err.Error()})
			continue
		}
		for field, value := range map[string]domain.Dec{
			"caneAvailable": r.CaneAvailable, "caneDelivered": r.CaneDelivered,
			"caneAccepted": r.CaneAccepted, "caneRejected": r.CaneRejected,
			"caneDiverted": r.CaneDiverted, "caneCrushed": r.CaneCrushed,
			"crushRateTph": r.CrushRateTPH,
		} {
			if value.IsNegative() {
				issues = append(issues, RowIssue{Row: i, Field: field, Code: "NEGATIVE",
					Message: fmt.Sprintf("%s cannot be negative", field)})
			}
		}
		if r.AvailableHrs.GreaterThan(domain.DI(24)) || r.AvailableHrs.IsNegative() {
			issues = append(issues, RowIssue{Row: i, Field: "availableHours", Code: "OUT_OF_RANGE",
				Message: "available hours must be between 0 and 24"})
		}
		if r.StoppageHrs.GreaterThan(r.AvailableHrs) {
			issues = append(issues, RowIssue{Row: i, Field: "stoppageHours", Code: "OUT_OF_RANGE",
				Message: "stoppage hours cannot exceed the available hours"})
		}
		if hasIssueForRow(issues, i) {
			continue
		}
		valid = append(valid, r)
	}

	return commitRowsGeneric(p, ctx, versionID, "daily_cane_plan", len(rows), valid, issues, opts,
		func(tx store.Store, rows []domain.DailyCanePlan) error {
			_, err := tx.Planning().UpsertCane(ctx, rows, caller.Username)
			return err
		})
}

// ---------------------------------------------------------------------------
// Production
// ---------------------------------------------------------------------------

// UpsertProduction writes daily production rows, planned or actual.
func (p *Planning) UpsertProduction(ctx context.Context, versionID string, rows []domain.DailyProductPlan,
	opts UpsertOptions) (UpsertResult, error) {

	caller := auth.FromContext(ctx)
	version, season, err := p.writeTarget(ctx, versionID)
	if err != nil {
		return UpsertResult{}, err
	}

	var issues []RowIssue
	valid := make([]domain.DailyProductPlan, 0, len(rows))
	for i, r := range rows {
		r.VersionID = versionID
		if r.FactoryID == "" {
			r.FactoryID = season.FactoryID
		}
		if bad := p.checkRow(caller, version, i, r.BusinessDate, r.Series, permForSeries(r.Series, domain.PermActualProduce)); bad != nil {
			issues = append(issues, *bad)
			continue
		}
		if r.ProductID == "" {
			issues = append(issues, RowIssue{Row: i, Field: "productId", Code: "REQUIRED",
				Message: "every production row needs a product"})
			continue
		}
		for field, value := range map[string]domain.Dec{
			"quantity": r.Quantity, "remeltInput": r.RemeltInput, "processLoss": r.ProcessLoss,
			"rework": r.Rework, "rejected": r.Rejected, "holdQty": r.HoldQty,
		} {
			if value.IsNegative() {
				issues = append(issues, RowIssue{Row: i, Field: field, Code: "NEGATIVE",
					Message: fmt.Sprintf("%s cannot be negative", field)})
			}
		}
		if hasIssueForRow(issues, i) {
			continue
		}
		valid = append(valid, r)
	}

	return commitRowsGeneric(p, ctx, versionID, "daily_product_plan", len(rows), valid, issues, opts,
		func(tx store.Store, rows []domain.DailyProductPlan) error {
			_, err := tx.Planning().UpsertProducts(ctx, rows, caller.Username)
			return err
		})
}

// ---------------------------------------------------------------------------
// Shipments
// ---------------------------------------------------------------------------

// UpsertShipments writes daily shipment rows, planned or actual.
func (p *Planning) UpsertShipments(ctx context.Context, versionID string, rows []domain.DailyShipmentPlan,
	opts UpsertOptions) (UpsertResult, error) {

	caller := auth.FromContext(ctx)
	version, _, err := p.writeTarget(ctx, versionID)
	if err != nil {
		return UpsertResult{}, err
	}

	var issues []RowIssue
	valid := make([]domain.DailyShipmentPlan, 0, len(rows))
	for i, r := range rows {
		r.VersionID = versionID
		if bad := p.checkRow(caller, version, i, r.BusinessDate, r.Series, permForSeries(r.Series, domain.PermActualShip)); bad != nil {
			issues = append(issues, *bad)
			continue
		}
		if r.ProductID == "" || r.ChannelID == "" {
			issues = append(issues, RowIssue{Row: i, Field: "channelId", Code: "REQUIRED",
				Message: "a shipment row needs both a product and a channel"})
			continue
		}
		if r.Quantity.IsNegative() {
			issues = append(issues, RowIssue{Row: i, Field: "quantity", Code: "NEGATIVE",
				Message: "the shipment quantity cannot be negative"})
			continue
		}
		valid = append(valid, r)
	}

	return commitRowsGeneric(p, ctx, versionID, "daily_shipment_plan", len(rows), valid, issues, opts,
		func(tx store.Store, rows []domain.DailyShipmentPlan) error {
			_, err := tx.Planning().UpsertShipments(ctx, rows, caller.Username)
			return err
		})
}

// ---------------------------------------------------------------------------
// Storage
// ---------------------------------------------------------------------------

// UpsertStorage writes stock ledger rows and rebuilds the running balances of
// every affected warehouse and product, so the ledger stays continuous.
func (p *Planning) UpsertStorage(ctx context.Context, versionID string, rows []domain.DailyStoragePlan,
	opts UpsertOptions) (UpsertResult, error) {

	caller := auth.FromContext(ctx)
	version, _, err := p.writeTarget(ctx, versionID)
	if err != nil {
		return UpsertResult{}, err
	}

	var issues []RowIssue
	valid := make([]domain.DailyStoragePlan, 0, len(rows))
	for i, r := range rows {
		r.VersionID = versionID
		if bad := p.checkRow(caller, version, i, r.BusinessDate, r.Series, permForSeries(r.Series, domain.PermActualStock)); bad != nil {
			issues = append(issues, *bad)
			continue
		}
		if r.WarehouseID == "" || r.ProductID == "" {
			issues = append(issues, RowIssue{Row: i, Field: "warehouseId", Code: "REQUIRED",
				Message: "a stock row needs both a warehouse and a product"})
			continue
		}
		for field, value := range map[string]domain.Dec{
			"productionReceipt": r.ProductionReceipt, "transferIn": r.TransferIn,
			"transferOut": r.TransferOut, "repackIn": r.RepackIn, "repackOut": r.RepackOut,
			"remeltIssue": r.RemeltIssue, "shipmentQty": r.ShipmentQty,
			"processLoss": r.ProcessLoss, "holdQty": r.HoldQty,
		} {
			if value.IsNegative() {
				issues = append(issues, RowIssue{Row: i, Field: field, Code: "NEGATIVE",
					Message: fmt.Sprintf("%s is a movement quantity and cannot be negative; "+
						"use adjustment for a correction downwards", field)})
			}
		}
		if hasIssueForRow(issues, i) {
			continue
		}
		valid = append(valid, r)
	}

	result, err := commitRowsGeneric(p, ctx, versionID, "daily_storage_plan", len(rows), valid, issues, opts,
		func(tx store.Store, rows []domain.DailyStoragePlan) error {
			if _, err := tx.Planning().UpsertStorage(ctx, rows, caller.Username); err != nil {
				return err
			}
			return p.rebuildBalances(ctx, tx, versionID, affectedStores(rows), caller.Username)
		})
	return result, err
}

// affectedStores lists the distinct warehouse and product pairs in a batch.
func affectedStores(rows []domain.DailyStoragePlan) [][2]string {
	seen := map[[2]string]bool{}
	var out [][2]string
	for _, r := range rows {
		key := [2]string{r.WarehouseID, r.ProductID}
		if !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	return out
}

// rebuildBalances recalculates the beginning and ending balances for the given
// stores across the whole version. Editing one day changes every day after it,
// so the recalculation has to run to the end of the season.
func (p *Planning) rebuildBalances(ctx context.Context, tx store.Store, versionID string,
	stores [][2]string, actor string) error {

	for _, key := range stores {
		warehouseID, productID := key[0], key[1]
		wh, err := tx.MasterData().Warehouses().Get(ctx, warehouseID)
		if err != nil {
			return err
		}
		rows, err := tx.Planning().ListStorage(ctx, store.PlanFilter{
			VersionIDs:   []string{versionID},
			WarehouseIDs: []string{warehouseID},
			ProductIDs:   []string{productID},
		})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			continue
		}
		sortStorage(rows)

		movements := make([]domain.LedgerMovement, len(rows))
		for i, r := range rows {
			movements[i] = domain.LedgerMovement{
				Date: r.BusinessDate, ProductionReceipt: r.ProductionReceipt,
				TransferIn: r.TransferIn, RepackIn: r.RepackIn, Adjustment: r.Adjustment,
				RemeltIssue: r.RemeltIssue, ShipmentQty: r.ShipmentQty,
				TransferOut: r.TransferOut, RepackOut: r.RepackOut, ProcessLoss: r.ProcessLoss,
				HoldQty: r.HoldQty, PhysicalBalance: r.PhysicalBalance,
			}
		}
		days := domain.RollLedger(wh.OpeningBalance, wh.UsableCapacity(), movements)
		for i := range rows {
			rows[i].BeginningBalance = days[i].BeginningBalance
			rows[i].EndingBalance = days[i].EndingBalance
		}
		if _, err := tx.Planning().UpsertStorage(ctx, rows, actor); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// writeTarget loads the version and season for a write and checks the scope.
func (p *Planning) writeTarget(ctx context.Context, versionID string) (domain.PlanVersion, domain.Season, error) {
	version, err := p.versionInScope(ctx, versionID)
	if err != nil {
		return domain.PlanVersion{}, domain.Season{}, err
	}
	season, err := p.store.Planning().GetSeason(ctx, version.SeasonID)
	if err != nil {
		return domain.PlanVersion{}, domain.Season{}, err
	}
	return version, season, nil
}

// permForSeries returns the permission a row needs: planned values need the
// planning permission, actuals need the permission for that kind of actual.
func permForSeries(series domain.Series, actualPerm string) string {
	if series == domain.SeriesActual {
		return actualPerm
	}
	return domain.PermPlanWrite
}

// checkRow applies the checks common to every daily row: a valid date, the
// permission for the series, and the version's locking rules.
func (p *Planning) checkRow(caller auth.Principal, version domain.PlanVersion, row int,
	date domain.BusinessDate, series domain.Series, permission string) *RowIssue {

	if series == "" {
		series = domain.SeriesPlan
	}
	if series != domain.SeriesPlan && series != domain.SeriesActual {
		return &RowIssue{Row: row, Field: "series", Code: "INVALID",
			Message: fmt.Sprintf("%q is not a known series; use PLAN or ACTUAL", series)}
	}
	if !date.Valid() {
		return &RowIssue{Row: row, Field: "businessDate", Code: "INVALID_DATE",
			Message: fmt.Sprintf("%q is not a valid ISO date", date)}
	}
	if err := caller.Require(permission); err != nil {
		return &RowIssue{Row: row, Field: "series", Code: "FORBIDDEN", Message: err.Error()}
	}
	if err := domain.CheckWritable(version, date, series); err != nil {
		code := "LOCKED"
		if !isLocked(err) {
			code = "INVALID_SERIES"
		}
		return &RowIssue{Row: row, Field: "businessDate", Code: code, Message: err.Error()}
	}
	return nil
}

func isLocked(err error) bool { return errors.Is(err, domain.ErrLocked) }

func hasIssueForRow(issues []RowIssue, row int) bool {
	for _, i := range issues {
		if i.Row == row {
			return true
		}
	}
	return false
}

// commitRows applies the all-or-nothing rule and writes the accepted rows in a
// single transaction together with the audit record.
func commitRowsGeneric[T any](p *Planning, ctx context.Context, versionID, entity string,
	total int, valid []T, issues []RowIssue, opts UpsertOptions,
	write func(store.Store, []T) error) (UpsertResult, error) {

	result := UpsertResult{Accepted: len(valid), Rejected: total - len(valid), Issues: issues}
	if len(issues) > 0 && !opts.Partial {
		result.Accepted = 0
		result.Rejected = total
		return result, &domain.ValidationError{Errors: toFieldErrors(issues)}
	}
	if len(valid) == 0 {
		return result, nil
	}

	err := p.store.InTx(ctx, func(tx store.Store) error {
		if err := write(tx, valid); err != nil {
			return err
		}
		return p.audit(ctx, tx, auditEntry{
			action: "UPSERT", entity: entity, entityID: versionID,
			reason: fmt.Sprintf("%d rows accepted, %d rejected", result.Accepted, result.Rejected),
		})
	})
	if err != nil {
		return UpsertResult{}, err
	}
	return result, nil
}

func toFieldErrors(issues []RowIssue) []domain.FieldError {
	out := make([]domain.FieldError, len(issues))
	for i, is := range issues {
		row := is.Row
		out[i] = domain.FieldError{Row: &row, Field: is.Field, Code: is.Code, Message: is.Message}
	}
	return out
}
