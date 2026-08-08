package service

import (
	"context"
	"sort"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Materials calculates packaging material requirements from the production
// plan (section 8 of the specification).
type Materials struct {
	store    store.Store
	planning *Planning
}

// NewMaterials builds the service.
func NewMaterials(s store.Store, p *Planning) *Materials {
	return &Materials{store: s, planning: p}
}

// MaterialRequirement is one material's requirement over the horizon.
type MaterialRequirement struct {
	MaterialID     string              `json:"materialId"`
	MaterialCode   string              `json:"materialCode"`
	MaterialName   string              `json:"materialName"`
	UOM            string              `json:"uom"`
	GrossRequired  domain.Dec          `json:"grossRequirement"`
	SafetyStock    domain.Dec          `json:"safetyStock"`
	OnHand         domain.Dec          `json:"available"`
	OnOrder        domain.Dec          `json:"openPurchase"`
	Shortage       domain.Dec          `json:"shortage"`
	PurchaseQty    domain.Dec          `json:"purchaseRequirement"`
	RequiredBy     domain.BusinessDate `json:"requiredBy,omitempty"`
	SuggestedOrder domain.BusinessDate `json:"suggestedOrderDate,omitempty"`
	LeadTimeDays   int                 `json:"leadTimeDays"`
	Severity       domain.Severity     `json:"severity"`
	// Sources shows which packed products drive the requirement, so a buyer can
	// see why the number is what it is.
	Sources []RequirementSource `json:"sources"`
}

// RequirementSource is one packaging type's contribution.
type RequirementSource struct {
	PackagingCode string     `json:"packagingCode"`
	PackagingName string     `json:"packagingName"`
	PackedTons    domain.Dec `json:"packedTons"`
	Packages      int64      `json:"packages"`
	Quantity      domain.Dec `json:"quantity"`
}

// RequirementsRequest selects the plan and horizon.
type RequirementsRequest struct {
	VersionID string              `json:"versionId"`
	From      domain.BusinessDate `json:"from,omitempty"`
	To        domain.BusinessDate `json:"to,omitempty"`
}

// Requirements calculates packaging needs for a plan version.
//
// The chain is: planned packed tons -> packages (net weight and scrap
// allowance) -> materials (the primary bag plus the packaging bill of
// materials) -> purchase requirement against stock and open orders.
func (m *Materials) Requirements(ctx context.Context, req RequirementsRequest) ([]MaterialRequirement, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermMaterialsRead); err != nil {
		return nil, err
	}
	if _, err := m.planning.GetVersion(ctx, req.VersionID); err != nil {
		return nil, err
	}

	rows, err := m.store.Planning().ListProducts(ctx, store.PlanFilter{
		VersionIDs: []string{req.VersionID}, From: req.From, To: req.To, Series: domain.SeriesPlan,
	})
	if err != nil {
		return nil, err
	}

	packagingPage, err := m.store.MasterData().PackagingTypes().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, err
	}
	packaging := map[string]domain.PackagingType{}
	for _, p := range packagingPage.Items {
		packaging[p.ID] = p
	}
	materialsPage, err := m.store.MasterData().Materials().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, err
	}
	materials := map[string]domain.Material{}
	for _, mat := range materialsPage.Items {
		materials[mat.ID] = mat
	}

	// Tonnage and earliest required date per packaging type.
	tonsByPackaging := map[string]domain.Dec{}
	firstDate := map[string]domain.BusinessDate{}
	for _, r := range rows {
		if r.PackagingID == "" || r.Quantity.LessThanOrEqual(domain.Zero) {
			continue
		}
		tonsByPackaging[r.PackagingID] = tonsByPackaging[r.PackagingID].Add(r.Quantity)
		if d, ok := firstDate[r.PackagingID]; !ok || r.BusinessDate < d {
			firstDate[r.PackagingID] = r.BusinessDate
		}
	}

	// Accumulate per material.
	acc := map[string]*MaterialRequirement{}
	need := func(mat domain.Material) *MaterialRequirement {
		if r, ok := acc[mat.ID]; ok {
			return r
		}
		r := &MaterialRequirement{
			MaterialID: mat.ID, MaterialCode: mat.Code, MaterialName: mat.Name,
			UOM: mat.UOM, SafetyStock: mat.SafetyStock, OnHand: mat.OnHand,
			OnOrder: mat.OnOrder, LeadTimeDays: mat.LeadTimeDays,
		}
		acc[mat.ID] = r
		return r
	}

	// The bill of materials covers everything that is not the primary bag: the
	// liner inside the jumbo bag, the thread that sews it, the label on it.
	// Without it a requirement report says the factory needs bags and nothing
	// else, which is the kind of answer that stops a line at two in the morning.
	bom, err := m.store.MasterData().ListPackagingBOM(ctx, "")
	if err != nil {
		return nil, err
	}
	componentsOf := map[string][]domain.PackagingBOMLine{}
	for _, line := range bom {
		if !line.Active {
			continue
		}
		componentsOf[line.PackagingID] = append(componentsOf[line.PackagingID], line)
	}

	for packagingID, tons := range tonsByPackaging {
		pack, ok := packaging[packagingID]
		if !ok {
			continue
		}

		// The package count is set by the bag, so it is worked out from the bag's
		// scrap rate and then shared by every component: a bag that tears is a
		// bag that had a liner in it.
		primary, hasPrimary := materials[pack.MaterialID]
		packages := domain.RequiredPackages(tons, pack.NetWeightKg, primary.ScrapPct)
		if packages <= 0 {
			continue
		}

		record := func(mat domain.Material, qty domain.Dec) {
			r := need(mat)
			r.GrossRequired = r.GrossRequired.Add(qty)
			r.Sources = append(r.Sources, RequirementSource{
				PackagingCode: pack.Code, PackagingName: pack.Name,
				PackedTons: domain.RoundQty(tons), Packages: packages, Quantity: qty,
			})
			if d := firstDate[packagingID]; d != "" && (r.RequiredBy == "" || d < r.RequiredBy) {
				r.RequiredBy = d
			}
		}

		if hasPrimary && pack.MaterialID != "" {
			record(primary, domain.DI(packages))
		}
		for _, line := range componentsOf[packagingID] {
			component, ok := materials[line.MaterialID]
			if !ok {
				continue
			}
			record(component, domain.ComponentQuantity(packages, line.QtyPerPackage, component.ScrapPct))
		}
	}

	out := make([]MaterialRequirement, 0, len(acc))
	for _, r := range acc {
		r.PurchaseQty = domain.PurchaseRequirement(r.GrossRequired, r.SafetyStock, r.OnHand, r.OnOrder)
		r.Shortage = domain.ClampNonNegative(
			domain.RoundQty(r.GrossRequired.Sub(r.OnHand).Sub(r.OnOrder)))
		if r.RequiredBy != "" && r.LeadTimeDays > 0 {
			r.SuggestedOrder = domain.SuggestedOrderDate(r.RequiredBy, r.LeadTimeDays)
		}
		switch {
		case r.Shortage.GreaterThan(domain.Zero):
			// Coverage runs out before the material is needed: production stops.
			r.Severity = domain.SeverityError
		case r.PurchaseQty.GreaterThan(domain.Zero):
			// Covered, but the safety stock would be eaten into.
			r.Severity = domain.SeverityWarning
		default:
			r.Severity = domain.SeveritySuccess
		}
		sort.Slice(r.Sources, func(i, j int) bool { return r.Sources[i].PackagingCode < r.Sources[j].PackagingCode })
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MaterialCode < out[j].MaterialCode })
	return out, nil
}
