package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// CompareRequest selects two versions and the dimension to compare them on.
type CompareRequest struct {
	BaseVersionID  string              `json:"baseVersionId"`
	OtherVersionID string              `json:"otherVersionId"`
	From           domain.BusinessDate `json:"from,omitempty"`
	To             domain.BusinessDate `json:"to,omitempty"`
	// Dimension is one of DATE, PRODUCT, WAREHOUSE, CHANNEL or PROCESS.
	Dimension string `json:"dimension"`
}

// CompareRow is one line of a version comparison.
type CompareRow struct {
	Key        string     `json:"key"`
	Label      string     `json:"label"`
	Measure    string     `json:"measure"`
	BaseValue  domain.Dec `json:"baseValue"`
	OtherValue domain.Dec `json:"otherValue"`
	Delta      domain.Dec `json:"delta"`
	DeltaPct   domain.Dec `json:"deltaPct"`
}

// CompareResult is the whole comparison.
type CompareResult struct {
	Base      domain.PlanVersion `json:"baseVersion"`
	Other     domain.PlanVersion `json:"otherVersion"`
	Dimension string             `json:"dimension"`
	Rows      []CompareRow       `json:"rows"`
	Totals    []CompareRow       `json:"totals"`
	// AssumptionDeltas is usually the most useful part of a scenario
	// comparison: it says what was changed to produce the difference.
	AssumptionDeltas []CompareRow `json:"assumptionDeltas"`
}

// Compare produces a side-by-side comparison of two plan versions.
func (p *Planning) Compare(ctx context.Context, req CompareRequest) (CompareResult, error) {
	base, err := p.versionInScope(ctx, req.BaseVersionID)
	if err != nil {
		return CompareResult{}, err
	}
	other, err := p.versionInScope(ctx, req.OtherVersionID)
	if err != nil {
		return CompareResult{}, err
	}
	if base.SeasonID != other.SeasonID {
		return CompareResult{}, fmt.Errorf(
			"%w: versions from different seasons cannot be compared", domain.ErrValidation)
	}
	if req.Dimension == "" {
		req.Dimension = "DATE"
	}

	result := CompareResult{Base: base, Other: other, Dimension: req.Dimension}

	baseValues, err := p.compareValues(ctx, base, req)
	if err != nil {
		return CompareResult{}, err
	}
	otherValues, err := p.compareValues(ctx, other, req)
	if err != nil {
		return CompareResult{}, err
	}

	labels, err := p.compareLabels(ctx, req.Dimension)
	if err != nil {
		return CompareResult{}, err
	}

	type cell struct{ key, measure string }
	keys := map[cell]bool{}
	for k := range baseValues {
		keys[cell{k.key, k.measure}] = true
	}
	for k := range otherValues {
		keys[cell{k.key, k.measure}] = true
	}

	for k := range keys {
		b := baseValues[compareKey{k.key, k.measure}]
		o := otherValues[compareKey{k.key, k.measure}]
		label := labels[k.key]
		if label == "" {
			label = k.key
		}
		result.Rows = append(result.Rows, CompareRow{
			Key: k.key, Label: label, Measure: k.measure,
			BaseValue: domain.RoundQty(b), OtherValue: domain.RoundQty(o),
			Delta:    domain.RoundQty(o.Sub(b)),
			DeltaPct: domain.RoundPct(domain.SafePct(o.Sub(b), b)),
		})
	}
	sort.Slice(result.Rows, func(i, j int) bool {
		if result.Rows[i].Measure != result.Rows[j].Measure {
			return result.Rows[i].Measure < result.Rows[j].Measure
		}
		return result.Rows[i].Key < result.Rows[j].Key
	})

	// Totals per measure.
	totals := map[string][2]domain.Dec{}
	for _, r := range result.Rows {
		t := totals[r.Measure]
		totals[r.Measure] = [2]domain.Dec{t[0].Add(r.BaseValue), t[1].Add(r.OtherValue)}
	}
	for measure, t := range totals {
		result.Totals = append(result.Totals, CompareRow{
			Key: "TOTAL", Label: "Total", Measure: measure,
			BaseValue: domain.RoundQty(t[0]), OtherValue: domain.RoundQty(t[1]),
			Delta:    domain.RoundQty(t[1].Sub(t[0])),
			DeltaPct: domain.RoundPct(domain.SafePct(t[1].Sub(t[0]), t[0])),
		})
	}
	sort.Slice(result.Totals, func(i, j int) bool { return result.Totals[i].Measure < result.Totals[j].Measure })

	result.AssumptionDeltas, err = p.compareAssumptions(ctx, base.ID, other.ID)
	if err != nil {
		return CompareResult{}, err
	}
	return result, nil
}

type compareKey struct{ key, measure string }

// compareValues aggregates one version onto the requested dimension.
func (p *Planning) compareValues(ctx context.Context, v domain.PlanVersion, req CompareRequest) (map[compareKey]domain.Dec, error) {
	series := domain.SeriesPlan
	if v.PlanType == domain.PlanTypeActual {
		series = domain.SeriesActual
	}
	f := store.PlanFilter{VersionIDs: []string{v.ID}, From: req.From, To: req.To, Series: series}
	out := map[compareKey]domain.Dec{}
	add := func(key, measure string, value domain.Dec) {
		k := compareKey{key, measure}
		out[k] = out[k].Add(value)
	}

	switch req.Dimension {
	case "DATE", "PROCESS":
		cane, err := p.store.Planning().ListCane(ctx, f)
		if err != nil {
			return nil, err
		}
		products, err := p.store.Planning().ListProducts(ctx, f)
		if err != nil {
			return nil, err
		}
		shipments, err := p.store.Planning().ListShipments(ctx, f)
		if err != nil {
			return nil, err
		}
		for _, r := range cane {
			key := string(r.BusinessDate)
			if req.Dimension == "PROCESS" {
				key = string(domain.StageCane)
			}
			add(key, "caneCrushed", r.CaneCrushed)
		}
		for _, r := range products {
			key := string(r.BusinessDate)
			if req.Dimension == "PROCESS" {
				key = string(domain.StageRefining)
			}
			add(key, "production", r.Quantity)
			add(key, "remeltInput", r.RemeltInput)
		}
		for _, r := range shipments {
			key := string(r.BusinessDate)
			if req.Dimension == "PROCESS" {
				key = string(domain.StageShipment)
			}
			add(key, "shipment", r.Quantity)
		}

	case "PRODUCT":
		products, err := p.store.Planning().ListProducts(ctx, f)
		if err != nil {
			return nil, err
		}
		for _, r := range products {
			add(r.ProductID, "production", r.Quantity)
		}

	case "WAREHOUSE":
		storageRows, err := p.store.Planning().ListStorage(ctx, f)
		if err != nil {
			return nil, err
		}
		peak := map[string]domain.Dec{}
		for _, r := range storageRows {
			add(r.WarehouseID, "receipts", r.ProductionReceipt.Add(r.TransferIn))
			add(r.WarehouseID, "issues", r.ShipmentQty.Add(r.RemeltIssue).Add(r.TransferOut))
			if r.EndingBalance.GreaterThan(peak[r.WarehouseID]) {
				peak[r.WarehouseID] = r.EndingBalance
			}
		}
		for wh, value := range peak {
			add(wh, "peakBalance", value)
		}

	case "CHANNEL":
		shipments, err := p.store.Planning().ListShipments(ctx, f)
		if err != nil {
			return nil, err
		}
		for _, r := range shipments {
			add(r.ChannelID, "shipment", r.Quantity)
		}

	default:
		return nil, fmt.Errorf("%w: %q is not a comparable dimension; use DATE, PROCESS, PRODUCT, WAREHOUSE or CHANNEL",
			domain.ErrValidation, req.Dimension)
	}
	return out, nil
}

// compareLabels resolves ids to readable names for the dimension.
func (p *Planning) compareLabels(ctx context.Context, dimension string) (map[string]string, error) {
	out := map[string]string{}
	switch dimension {
	case "PRODUCT":
		page, err := p.store.MasterData().Products().List(ctx, store.ListOptions{Top: 1000})
		if err != nil {
			return nil, err
		}
		for _, x := range page.Items {
			out[x.ID] = x.Code + " " + x.Name
		}
	case "WAREHOUSE":
		page, err := p.store.MasterData().Warehouses().List(ctx, store.ListOptions{Top: 1000})
		if err != nil {
			return nil, err
		}
		for _, x := range page.Items {
			out[x.ID] = x.Code + " " + x.Name
		}
	case "CHANNEL":
		page, err := p.store.MasterData().Channels().List(ctx, store.ListOptions{Top: 1000})
		if err != nil {
			return nil, err
		}
		for _, x := range page.Items {
			out[x.ID] = x.Code + " " + x.Name
		}
	}
	return out, nil
}

// compareAssumptions lists the assumptions that differ between two versions.
func (p *Planning) compareAssumptions(ctx context.Context, baseID, otherID string) ([]CompareRow, error) {
	baseRows, err := p.store.Planning().ListAssumptions(ctx, baseID)
	if err != nil {
		return nil, err
	}
	otherRows, err := p.store.Planning().ListAssumptions(ctx, otherID)
	if err != nil {
		return nil, err
	}
	base := map[string]domain.PlanAssumption{}
	for _, a := range baseRows {
		base[a.Code] = a
	}
	other := map[string]domain.PlanAssumption{}
	for _, a := range otherRows {
		other[a.Code] = a
	}

	codes := map[string]bool{}
	for c := range base {
		codes[c] = true
	}
	for c := range other {
		codes[c] = true
	}

	var out []CompareRow
	for code := range codes {
		b, o := base[code].Value, other[code].Value
		if b.Equal(o) {
			continue
		}
		label := base[code].Description
		if label == "" {
			label = other[code].Description
		}
		if label == "" {
			label = code
		}
		out = append(out, CompareRow{
			Key: code, Label: label, Measure: "assumption",
			BaseValue: b, OtherValue: o,
			Delta:    o.Sub(b),
			DeltaPct: domain.RoundPct(domain.SafePct(o.Sub(b), b)),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}
