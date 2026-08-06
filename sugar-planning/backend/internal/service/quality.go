package service

import (
	"context"
	"fmt"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// This file is the laboratory and the maintenance calendar: parameters and
// their effective-dated specifications, samples and the results judged against
// them, the holds a failure places on stock, and the outages that take crushing
// days out of the plan.

// ---------------------------------------------------------------------------
// Parameters and specifications
// ---------------------------------------------------------------------------

// ListParameters returns the measurable properties the site has defined.
func (e *Execution) ListParameters(ctx context.Context) ([]domain.QualityParameter, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermMasterDataRead); err != nil {
		return nil, err
	}
	return e.store.Execution().ListParameters(ctx)
}

// SaveParameter creates or updates a quality parameter.
func (e *Execution) SaveParameter(ctx context.Context, p domain.QualityParameter) (domain.QualityParameter, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermMasterDataWrite); err != nil {
		return domain.QualityParameter{}, err
	}
	verr := &domain.ValidationError{}
	if p.Code == "" {
		verr.Add("code", "REQUIRED", "the parameter needs a code, for example POL")
	}
	if p.Name == "" {
		verr.Add("name", "REQUIRED", "the parameter needs a name")
	}
	if err := verr.OrNil(); err != nil {
		return domain.QualityParameter{}, err
	}

	var saved domain.QualityParameter
	err := e.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Execution().SaveParameter(ctx, p, caller.Username)
		if err != nil {
			return err
		}
		return e.audit(ctx, tx, auditEntry{
			action: actionFor(p.ID == ""), entity: "quality_parameter",
			entityID: saved.ID, after: saved,
		})
	})
	return saved, err
}

// SpecsFor returns the specifications that judge a product on a date.
func (e *Execution) SpecsFor(ctx context.Context, productID string, on domain.BusinessDate) ([]domain.QualitySpec, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermMasterDataRead); err != nil {
		return nil, err
	}
	if !on.Valid() {
		return nil, fmt.Errorf("%w: %q is not a valid date", domain.ErrValidation, on)
	}
	return e.store.Execution().SpecsFor(ctx, productID, on)
}

// SaveSpec creates or updates a specification.
//
// Limits are effective dated, so tightening one does not retrospectively fail
// last season's production. Changing a limit that is already in force is done
// by ending the current specification and starting a new one, which is why the
// business key includes the start date.
func (e *Execution) SaveSpec(ctx context.Context, s domain.QualitySpec) (domain.QualitySpec, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermMasterDataWrite); err != nil {
		return domain.QualitySpec{}, err
	}

	verr := &domain.ValidationError{}
	if s.ProductID == "" {
		verr.Add("productId", "REQUIRED", "the specification needs a product")
	}
	if s.ParameterID == "" {
		verr.Add("parameterId", "REQUIRED", "the specification needs a parameter")
	}
	if !s.ValidFrom.Valid() {
		verr.Add("validFrom", "INVALID_DATE", "the specification needs a start date")
	}
	if s.ValidTo != "" && s.ValidTo < s.ValidFrom {
		verr.Add("validTo", "OUT_OF_RANGE", "the end date cannot be before the start date")
	}
	if s.LowerLimit != nil && s.UpperLimit != nil && s.UpperLimit.LessThan(*s.LowerLimit) {
		verr.Add("upperLimit", "OUT_OF_RANGE", "the upper limit is below the lower limit")
	}
	if s.WarnLower != nil && s.LowerLimit != nil && s.WarnLower.LessThan(*s.LowerLimit) {
		verr.Add("warnLower", "OUT_OF_RANGE",
			"the lower warning sits below the hard lower limit, so it can never be reached")
	}
	if s.WarnUpper != nil && s.UpperLimit != nil && s.WarnUpper.GreaterThan(*s.UpperLimit) {
		verr.Add("warnUpper", "OUT_OF_RANGE",
			"the upper warning sits above the hard upper limit, so it can never be reached")
	}
	if s.LowerLimit == nil && s.UpperLimit == nil && s.WarnLower == nil && s.WarnUpper == nil {
		verr.Add("lowerLimit", "EMPTY",
			"a specification with no limits cannot fail anything; give it at least one limit")
	}
	if err := verr.OrNil(); err != nil {
		return domain.QualitySpec{}, err
	}

	var saved domain.QualitySpec
	err := e.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Execution().SaveSpec(ctx, s, caller.Username)
		if err != nil {
			return err
		}
		return e.audit(ctx, tx, auditEntry{
			action: actionFor(s.ID == ""), entity: "quality_spec", entityID: saved.ID, after: saved,
		})
	})
	return saved, err
}

// ---------------------------------------------------------------------------
// Samples and results
// ---------------------------------------------------------------------------

// SampleRequest records material taken for testing.
type SampleRequest struct {
	ProductID    string              `json:"productId"`
	BatchID      string              `json:"batchId,omitempty"`
	FactoryID    string              `json:"factoryId"`
	BusinessDate domain.BusinessDate `json:"businessDate"`
	ShiftID      string              `json:"shiftId,omitempty"`
	Comment      string              `json:"comment,omitempty"`
}

// CreateSample opens a sample for the laboratory to fill in.
func (e *Execution) CreateSample(ctx context.Context, req SampleRequest) (domain.QualitySample, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermQualityWrite); err != nil {
		return domain.QualitySample{}, err
	}
	if err := caller.RequireFactory(req.FactoryID); err != nil {
		return domain.QualitySample{}, err
	}

	verr := &domain.ValidationError{}
	if req.ProductID == "" {
		verr.Add("productId", "REQUIRED", "the sample needs a product")
	}
	if !req.BusinessDate.Valid() {
		verr.Add("businessDate", "INVALID_DATE", "the sample needs a valid business date")
	}
	if err := verr.OrNil(); err != nil {
		return domain.QualitySample{}, err
	}

	factory, err := e.store.MasterData().Factories().Get(ctx, req.FactoryID)
	if err != nil {
		return domain.QualitySample{}, err
	}
	year, err := yearOf(req.BusinessDate)
	if err != nil {
		return domain.QualitySample{}, err
	}

	var saved domain.QualitySample
	err = e.store.InTx(ctx, func(tx store.Store) error {
		sampleNo, err := tx.Execution().NextNumber(ctx, store.SeriesSample, factory.Code, year)
		if err != nil {
			return err
		}
		saved, err = tx.Execution().SaveSample(ctx, domain.QualitySample{
			SampleNo: sampleNo, ProductID: req.ProductID, BatchID: req.BatchID,
			FactoryID: req.FactoryID, BusinessDate: req.BusinessDate, ShiftID: req.ShiftID,
			LabUser: caller.Username, Status: "OPEN", Comment: req.Comment,
		}, caller.Username)
		if err != nil {
			return err
		}
		return e.audit(ctx, tx, auditEntry{
			action: "CREATE", entity: "quality_sample", entityID: saved.ID, after: saved,
		})
	})
	return saved, err
}

// ResultInput is one measurement the laboratory entered.
type ResultInput struct {
	ParameterID string     `json:"parameterId"`
	Value       domain.Dec `json:"value"`
	UOM         string     `json:"uom,omitempty"`
	Comment     string     `json:"comment,omitempty"`
}

// ResultsRequest is the laboratory sheet for one sample.
type ResultsRequest struct {
	Results []ResultInput `json:"results"`
	// Complete closes the sample. An incomplete sheet can be saved and picked
	// up later; only a complete one produces a verdict and, if it fails, a
	// hold.
	Complete bool `json:"complete"`
	// HoldWarehouse and HoldQuantity place the failing material on hold. A
	// failure with nowhere named is still recorded and still fails; it just
	// does not block any particular stock.
	HoldWarehouse string     `json:"holdWarehouse,omitempty"`
	HoldQuantity  domain.Dec `json:"holdQuantity,omitempty"`
}

// ResultsOutcome is what recording a sheet did.
type ResultsOutcome struct {
	Sample domain.QualitySample `json:"sample"`
	// Verdict is the worst of the results.
	Verdict domain.QualityStatus `json:"verdict"`
	Hold    *domain.QualityHold  `json:"hold,omitempty"`
	// Unspecified names the parameters that had no effective specification, so
	// the laboratory learns that a measurement it entered judged nothing.
	Unspecified []string `json:"unspecified,omitempty"`
}

// RecordResults judges a sheet of measurements against the specifications in
// force on the sample's business date and, when the sample fails, blocks the
// material.
//
// The limits are copied onto each result rather than pointed at, because a
// result has to keep saying what it was judged against even after somebody
// edits the specification.
func (e *Execution) RecordResults(ctx context.Context, sampleID string, req ResultsRequest) (ResultsOutcome, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermQualityWrite); err != nil {
		return ResultsOutcome{}, err
	}

	sample, err := e.store.Execution().GetSample(ctx, sampleID)
	if err != nil {
		return ResultsOutcome{}, err
	}
	if err := caller.RequireFactory(sample.FactoryID); err != nil {
		return ResultsOutcome{}, err
	}
	if sample.Status != "OPEN" {
		return ResultsOutcome{}, fmt.Errorf("%w: sample %s is %s and takes no more results",
			domain.ErrStateTransition, sample.SampleNo, sample.Status)
	}

	specs, err := e.store.Execution().SpecsFor(ctx, sample.ProductID, sample.BusinessDate)
	if err != nil {
		return ResultsOutcome{}, err
	}
	// Two specifications for the same parameter can both be in force when a
	// tighter limit was started without ending the older one. The later start
	// date wins, because that is the one somebody most recently decided on;
	// leaving it to the order the repository happened to return would make the
	// verdict depend on the storage engine.
	byParameter := map[string]domain.QualitySpec{}
	for _, s := range specs {
		if current, ok := byParameter[s.ParameterID]; ok && current.ValidFrom >= s.ValidFrom {
			continue
		}
		byParameter[s.ParameterID] = s
	}

	parameters, err := e.store.Execution().ListParameters(ctx)
	if err != nil {
		return ResultsOutcome{}, err
	}
	known := map[string]domain.QualityParameter{}
	for _, p := range parameters {
		known[p.ID] = p
	}

	verr := &domain.ValidationError{}
	outcome := ResultsOutcome{}
	results := make([]domain.QualityResult, 0, len(req.Results))
	seen := map[string]bool{}
	for i, r := range req.Results {
		parameter, ok := known[r.ParameterID]
		if !ok {
			verr.AddRow(i, "parameterId", "NOT_FOUND", "no such quality parameter")
			continue
		}
		if seen[r.ParameterID] {
			verr.AddRow(i, "parameterId", "DUPLICATE",
				fmt.Sprintf("%s is measured twice on this sheet", parameter.Code))
			continue
		}
		seen[r.ParameterID] = true

		uom := r.UOM
		if uom == "" {
			uom = parameter.UOM
		}
		result := domain.QualityResult{
			SampleID: sample.ID, ParameterID: r.ParameterID, Value: r.Value,
			UOM: uom, Comment: r.Comment, Status: domain.QualityPass,
		}
		spec, ok := byParameter[r.ParameterID]
		if !ok {
			// A measurement with no specification in force judges nothing. It
			// is still recorded - the number is real - but the laboratory is
			// told, because an unspecified parameter is a configuration gap and
			// silently passing it is how a limit goes years without being set.
			outcome.Unspecified = append(outcome.Unspecified, parameter.Code)
		} else {
			result.LowerLimit, result.UpperLimit = spec.LowerLimit, spec.UpperLimit
			result.Status = domain.EvaluateResult(r.Value, spec)
		}
		results = append(results, result)
	}
	if err := verr.OrNil(); err != nil {
		return ResultsOutcome{}, err
	}

	sample.Results = results
	outcome.Verdict = sample.Verdict()

	// Blocking the material is part of failing it, not a separate warehouse
	// act: the laboratory's permission is enough, exactly as a production
	// confirmation receipts its own yield without needing the keeper's.
	placingHold := req.Complete && outcome.Verdict == domain.QualityFail &&
		req.HoldWarehouse != "" && req.HoldQuantity.GreaterThan(domain.Zero)

	err = e.store.InTx(ctx, func(tx store.Store) error {
		if err := tx.Execution().SaveResults(ctx, sample.ID, results, caller.Username); err != nil {
			return err
		}
		if req.Complete {
			sample.Status = "COMPLETE"
		}
		stored := sample
		saved, err := tx.Execution().SaveSample(ctx, stored, caller.Username)
		if err != nil {
			return err
		}
		saved.Results = results
		outcome.Sample = saved

		if placingHold {
			hold, err := e.placeHold(ctx, tx, domain.QualityHold{
				WarehouseID: req.HoldWarehouse, ProductID: sample.ProductID,
				BatchID: sample.BatchID, SampleID: sample.ID,
				Quantity: domain.RoundQty(req.HoldQuantity), PlacedOn: sample.BusinessDate,
				Reason: fmt.Sprintf("sample %s failed", sample.SampleNo),
			})
			if err != nil {
				return err
			}
			outcome.Hold = &hold
		}

		return e.audit(ctx, tx, auditEntry{
			action: "RESULTS", entity: "quality_sample", entityID: sample.ID,
			after:  outcome.Sample,
			reason: fmt.Sprintf("verdict %s", outcome.Verdict),
		})
	})
	if err != nil {
		return ResultsOutcome{}, err
	}
	return outcome, nil
}

// ---------------------------------------------------------------------------
// Holds
// ---------------------------------------------------------------------------

// HoldRequest blocks a quantity of stock.
type HoldRequest struct {
	WarehouseID string              `json:"warehouseId"`
	ProductID   string              `json:"productId"`
	BatchID     string              `json:"batchId,omitempty"`
	SampleID    string              `json:"sampleId,omitempty"`
	Quantity    domain.Dec          `json:"quantity"`
	PlacedOn    domain.BusinessDate `json:"placedOn"`
	Reason      string              `json:"reason"`
}

// PlaceHold blocks stock from shipment and consumption.
func (e *Execution) PlaceHold(ctx context.Context, req HoldRequest) (domain.QualityHold, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermQualityWrite); err != nil {
		return domain.QualityHold{}, err
	}

	verr := &domain.ValidationError{}
	if req.WarehouseID == "" {
		verr.Add("warehouseId", "REQUIRED", "the hold needs a warehouse")
	}
	if req.ProductID == "" {
		verr.Add("productId", "REQUIRED", "the hold needs a product")
	}
	if req.Quantity.LessThanOrEqual(domain.Zero) {
		verr.Add("quantity", "NOT_POSITIVE", "the held quantity must be more than zero")
	}
	if !req.PlacedOn.Valid() {
		verr.Add("placedOn", "INVALID_DATE", "the hold needs a valid date")
	}
	if req.Reason == "" {
		verr.Add("reason", "REQUIRED", "a hold needs a reason; it is what the release is judged against")
	}
	if err := verr.OrNil(); err != nil {
		return domain.QualityHold{}, err
	}

	warehouse, err := e.store.MasterData().Warehouses().Get(ctx, req.WarehouseID)
	if err != nil {
		return domain.QualityHold{}, err
	}
	if err := caller.RequireFactory(warehouse.FactoryID); err != nil {
		return domain.QualityHold{}, err
	}

	var saved domain.QualityHold
	err = e.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = e.placeHold(ctx, tx, domain.QualityHold{
			WarehouseID: req.WarehouseID, ProductID: req.ProductID, BatchID: req.BatchID,
			SampleID: req.SampleID, Quantity: domain.RoundQty(req.Quantity),
			PlacedOn: req.PlacedOn, Reason: req.Reason,
		})
		if err != nil {
			return err
		}
		return e.audit(ctx, tx, auditEntry{
			action: "HOLD", entity: "quality_hold", entityID: saved.ID,
			after: saved, reason: req.Reason,
		})
	})
	if err != nil {
		return domain.QualityHold{}, err
	}
	return saved, nil
}

// placeHold records the hold and posts the HOLD document that moves the held
// quantity on the balance.
//
// Both happen together, so a hold on the quality record and a balance that
// still shows the sugar as available cannot drift apart: the posting is what
// makes StockPosition.Available refuse to ship it.
func (e *Execution) placeHold(ctx context.Context, tx store.Store, hold domain.QualityHold) (domain.QualityHold, error) {
	caller := auth.FromContext(ctx)

	warehouse, err := tx.MasterData().Warehouses().Get(ctx, hold.WarehouseID)
	if err != nil {
		return domain.QualityHold{}, err
	}
	product, err := tx.MasterData().Products().Get(ctx, hold.ProductID)
	if err != nil {
		return domain.QualityHold{}, err
	}

	if _, err := e.postChecked(ctx, tx, domain.InventoryDocument{
		DocType: domain.DocHold, BusinessDate: hold.PlacedOn, FactoryID: warehouse.FactoryID,
		Note: hold.Reason,
		Items: []domain.InventoryDocumentItem{{
			LineNo: 1, WarehouseID: hold.WarehouseID, ProductID: hold.ProductID,
			BatchID: hold.BatchID, Quantity: hold.Quantity, UOM: product.BaseUOM,
		}},
	}, domain.PostingOptions{}); err != nil {
		return domain.QualityHold{}, err
	}

	return tx.Execution().SaveHold(ctx, hold, caller.Username)
}

// ReleaseRequest frees held stock.
type ReleaseRequest struct {
	ReleasedOn domain.BusinessDate `json:"releasedOn,omitempty"`
	Reason     string              `json:"reason"`
	RowVersion int64               `json:"rowVersion"`
}

// ReleaseHold frees blocked stock.
//
// It sits behind quality:release rather than quality:write, so a site that
// wants the person who stops the sugar leaving to be a different person from
// the one who decides it may leave after all can arrange that by granting the
// two permissions to different roles. The shipped quality role holds both.
func (e *Execution) ReleaseHold(ctx context.Context, holdID string, req ReleaseRequest) (domain.QualityHold, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermQualityRelease); err != nil {
		return domain.QualityHold{}, err
	}

	hold, err := e.store.Execution().GetHold(ctx, holdID)
	if err != nil {
		return domain.QualityHold{}, err
	}
	if !hold.IsOpen() {
		return domain.QualityHold{}, fmt.Errorf("%w: the hold was already released on %s",
			domain.ErrValidation, hold.ReleasedOn)
	}
	if req.RowVersion != 0 && req.RowVersion != hold.RowVersion {
		return domain.QualityHold{}, fmt.Errorf("%w: the hold is at version %d, you have %d",
			domain.ErrConflict, hold.RowVersion, req.RowVersion)
	}
	if req.Reason == "" {
		return domain.QualityHold{}, fmt.Errorf(
			"%w: releasing a hold needs a reason for the audit trail", domain.ErrValidation)
	}

	warehouse, err := e.store.MasterData().Warehouses().Get(ctx, hold.WarehouseID)
	if err != nil {
		return domain.QualityHold{}, err
	}
	if err := caller.RequireFactory(warehouse.FactoryID); err != nil {
		return domain.QualityHold{}, err
	}
	product, err := e.store.MasterData().Products().Get(ctx, hold.ProductID)
	if err != nil {
		return domain.QualityHold{}, err
	}

	releasedOn := req.ReleasedOn
	if releasedOn == "" {
		releasedOn = domain.NewBusinessDate(e.now())
	}
	if releasedOn < hold.PlacedOn {
		return domain.QualityHold{}, fmt.Errorf(
			"%w: the release date is before the hold was placed", domain.ErrValidation)
	}

	before := hold
	hold.ReleasedOn, hold.ReleasedBy = releasedOn, caller.Username
	hold.Reason = joinReason(hold.Reason, "released: "+req.Reason)

	var saved domain.QualityHold
	err = e.store.InTx(ctx, func(tx store.Store) error {
		if _, err := e.postChecked(ctx, tx, domain.InventoryDocument{
			DocType: domain.DocRelease, BusinessDate: releasedOn, FactoryID: warehouse.FactoryID,
			Note: req.Reason,
			Items: []domain.InventoryDocumentItem{{
				LineNo: 1, WarehouseID: hold.WarehouseID, ProductID: hold.ProductID,
				BatchID: hold.BatchID, Quantity: before.Quantity, UOM: product.BaseUOM,
			}},
		}, domain.PostingOptions{}); err != nil {
			return err
		}
		var err error
		saved, err = tx.Execution().SaveHold(ctx, hold, caller.Username)
		if err != nil {
			return err
		}
		return e.audit(ctx, tx, auditEntry{
			action: "RELEASE_HOLD", entity: "quality_hold", entityID: saved.ID,
			before: before, after: saved, reason: req.Reason,
		})
	})
	if err != nil {
		return domain.QualityHold{}, err
	}
	return saved, nil
}

// ListHolds returns the holds on stock the caller may see.
func (e *Execution) ListHolds(ctx context.Context, f store.ExecutionFilter) ([]domain.QualityHold, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return nil, err
	}
	holds, err := e.store.Execution().ListHolds(ctx, f)
	if err != nil {
		return nil, err
	}
	md := e.store.MasterData()
	warehouses := map[string]domain.Warehouse{}
	out := holds[:0]
	for _, h := range holds {
		warehouse, ok := warehouses[h.WarehouseID]
		if !ok {
			if warehouse, err = md.Warehouses().Get(ctx, h.WarehouseID); err != nil {
				return nil, err
			}
			warehouses[h.WarehouseID] = warehouse
		}
		if caller.CanSeeFactory(warehouse.FactoryID) {
			out = append(out, h)
		}
	}
	return out, nil
}

// ListSamples returns the samples the caller may see.
func (e *Execution) ListSamples(ctx context.Context, f store.ExecutionFilter) (store.Page[domain.QualitySample], error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return store.Page[domain.QualitySample]{}, err
	}
	if err := e.requireScope(caller, f.FactoryID); err != nil {
		return store.Page[domain.QualitySample]{}, err
	}
	page, err := e.store.Execution().ListSamples(ctx, f)
	if err != nil {
		return page, err
	}
	filtered := page.Items[:0]
	for _, s := range page.Items {
		if caller.CanSeeFactory(s.FactoryID) {
			filtered = append(filtered, s)
		}
	}
	page.Items = filtered
	page.Count = len(filtered)
	return page, nil
}

// GetSample reads one sample with its results.
func (e *Execution) GetSample(ctx context.Context, id string) (domain.QualitySample, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return domain.QualitySample{}, err
	}
	s, err := e.store.Execution().GetSample(ctx, id)
	if err != nil {
		return domain.QualitySample{}, err
	}
	if err := caller.RequireFactory(s.FactoryID); err != nil {
		return domain.QualitySample{}, err
	}
	return s, nil
}

// ---------------------------------------------------------------------------
// Maintenance
// ---------------------------------------------------------------------------

// ListMaintenance returns the outage calendar.
func (e *Execution) ListMaintenance(ctx context.Context, f store.ExecutionFilter) ([]domain.MaintenanceWindow, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return nil, err
	}
	if err := e.requireScope(caller, f.FactoryID); err != nil {
		return nil, err
	}
	windows, err := e.store.Execution().ListMaintenance(ctx, f)
	if err != nil {
		return nil, err
	}
	out := windows[:0]
	for _, w := range windows {
		if caller.CanSeeFactory(w.FactoryID) {
			out = append(out, w)
		}
	}
	return out, nil
}

// SaveMaintenance creates or amends a maintenance window.
//
// Two permissions meet here. Writing a window at all needs downtime:write,
// which is the engineering role. Approving one needs plan:approve as well,
// because an approved window lengthens the campaign: the generator treats its
// days as non-working and the season's end date moves. An approver may
// therefore save a window without holding the engineering permission - that is
// the whole of what approving is - but nobody approves on the engineering
// permission alone.
func (e *Execution) SaveMaintenance(ctx context.Context, m domain.MaintenanceWindow) (domain.MaintenanceWindow, error) {
	caller := auth.FromContext(ctx)
	if !caller.Can(domain.PermDowntimeWrite) && !caller.Can(domain.PermPlanApprove) {
		return domain.MaintenanceWindow{}, caller.Require(domain.PermDowntimeWrite)
	}
	if err := caller.RequireFactory(m.FactoryID); err != nil {
		return domain.MaintenanceWindow{}, err
	}

	verr := &domain.ValidationError{}
	if !m.StartDate.Valid() {
		verr.Add("startDate", "INVALID_DATE", "the window needs a valid start date")
	}
	if !m.EndDate.Valid() {
		verr.Add("endDate", "INVALID_DATE", "the window needs a valid end date")
	}
	if m.StartDate.Valid() && m.EndDate.Valid() && m.EndDate < m.StartDate {
		verr.Add("endDate", "OUT_OF_RANGE", "the window ends before it starts")
	}
	if m.Description == "" {
		verr.Add("description", "REQUIRED", "say what the outage is for")
	}
	switch m.Status {
	case "", domain.MaintenancePlanned, domain.MaintenanceDone, domain.MaintenanceCancelled:
		// fine
	case domain.MaintenanceApproved:
		if err := caller.Require(domain.PermPlanApprove); err != nil {
			return domain.MaintenanceWindow{}, fmt.Errorf(
				"%w: approving a maintenance window removes crushing days from the plan and needs the %s permission",
				domain.ErrForbidden, domain.PermPlanApprove)
		}
		if m.ApprovedBy == "" {
			m.ApprovedBy = caller.Username
		}
	default:
		verr.Add("status", "INVALID", fmt.Sprintf("%q is not a maintenance status", m.Status))
	}
	if err := verr.OrNil(); err != nil {
		return domain.MaintenanceWindow{}, err
	}

	var before any
	if m.ID != "" {
		existing, err := e.findMaintenance(ctx, m.FactoryID, m.ID)
		if err != nil {
			return domain.MaintenanceWindow{}, err
		}
		before = existing
	}

	var saved domain.MaintenanceWindow
	err := e.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Execution().SaveMaintenance(ctx, m, caller.Username)
		if err != nil {
			return err
		}
		return e.audit(ctx, tx, auditEntry{
			action: actionFor(m.ID == ""), entity: "maintenance_window",
			entityID: saved.ID, before: before, after: saved,
		})
	})
	if err != nil {
		return domain.MaintenanceWindow{}, err
	}
	return saved, nil
}

// findMaintenance reads one window. The repository lists rather than gets, so
// the lookup is by factory and id.
func (e *Execution) findMaintenance(ctx context.Context, factoryID, id string) (domain.MaintenanceWindow, error) {
	windows, err := e.store.Execution().ListMaintenance(ctx, store.ExecutionFilter{FactoryID: factoryID})
	if err != nil {
		return domain.MaintenanceWindow{}, err
	}
	for _, w := range windows {
		if w.ID == id {
			return w, nil
		}
	}
	return domain.MaintenanceWindow{}, fmt.Errorf("%w: maintenance window %s", domain.ErrNotFound, id)
}
