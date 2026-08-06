package seed

import (
	"context"
	"fmt"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// A second company, so that data scope can actually be tested.
//
// The reference scenario is one company at one factory, which is enough to
// demonstrate the system and not enough to test the thing most worth testing.
// With a single tenant, "a planner at Kampong Speu cannot see another factory's
// plan" can only be checked against a factory that does not exist — which
// proves a caller scoped to nothing sees nothing, and says nothing at all about
// whether two real tenants are kept apart.
//
// This is the second tenant: a smaller mill on the other side of the country,
// with its own season, its own stores and its own plan. It is deliberately
// **not** part of the demonstration profile. Adding it there would change what
// `GET /seasons` returns and make the reference figures something a reader has
// to filter for rather than read; the demonstration stays quotable and the test
// system gets what it needs.
type TenantResult struct {
	CompanyID  string
	FactoryID  string
	SeasonID   string
	BudgetID   string
	Warehouses map[string]string
	Generated  service.GenerateResult
}

// SecondTenantCodes are the business keys the second tenant uses, so a test can
// name them without holding an id.
const (
	SecondCompanyCode = "BTB"
	SecondFactoryCode = "F2"
	SecondSeasonCode  = "2026-2027-BTB"
)

// LoadSecondTenant adds a second company and factory with a season of its own.
//
// It is idempotent on business keys, like the rest of the seed, and it reuses
// the shared master data - products, units, packaging, reason codes - because
// those are not factory-scoped and duplicating them would be inventing a
// difference the schema does not have.
func LoadSecondTenant(ctx context.Context, s store.Store, planning *service.Planning) (TenantResult, error) {
	var out TenantResult
	md := s.MasterData()

	// The seed principal for the *first* company is not enough here: creating a
	// company is a master-data act, and writing a plan for the second factory
	// needs scope over it. The principal is widened once the ids exist.
	setup := auth.WithPrincipal(ctx, seedPrincipal("", ""))

	company, err := upsert(setup, md.Companies(), SecondCompanyCode, domain.Company{
		Code: SecondCompanyCode, Name: "Battambang Cane Millers Ltd.",
		Currency: "USD", TimeZone: "Asia/Phnom_Penh", Validity: active(),
	})
	if err != nil {
		return out, fmt.Errorf("second company: %w", err)
	}
	out.CompanyID = company.ID

	factory, err := upsert(setup, md.Factories(), SecondFactoryCode, domain.Factory{
		CompanyID: company.ID, Code: SecondFactoryCode, Name: "Factory 2 - Battambang",
		TimeZone: "Asia/Phnom_Penh", Validity: active(),
	})
	if err != nil {
		return out, fmt.Errorf("second factory: %w", err)
	}
	out.FactoryID = factory.ID

	// From here everything is scoped to the second tenant.
	ctx = auth.WithPrincipal(ctx, seedPrincipal(company.ID, factory.ID))

	if _, err := upsert(ctx, md.Lines(), "MILL-2", domain.ProductionLine{
		FactoryID: factory.ID, Code: "MILL-2", Name: "Cane milling train 2",
		Stage: domain.StageCane, RatedTPH: domain.D("320"), Validity: active(),
	}); err != nil {
		return out, fmt.Errorf("second factory line: %w", err)
	}

	out.Warehouses = map[string]string{}
	for _, w := range []domain.Warehouse{
		{FactoryID: factory.ID, Code: "BTB-RAW", Name: "Battambang raw sugar store",
			StorageClass: domain.StorageRaw, CapacityTons: domain.D("18000"),
			UsablePct: domain.D("95"), Validity: active()},
		{FactoryID: factory.ID, Code: "BTB-FG", Name: "Battambang finished goods store",
			StorageClass: domain.StorageFinished, CapacityTons: domain.D("9000"),
			UsablePct: domain.D("95"), Validity: active()},
	} {
		saved, err := upsert(ctx, md.Warehouses(), w.Code, w)
		if err != nil {
			return out, fmt.Errorf("second factory warehouse %s: %w", w.Code, err)
		}
		out.Warehouses[w.Code] = saved.ID
	}

	season, err := findOrCreateSeason(ctx, planning, domain.Season{
		CompanyID: company.ID, FactoryID: factory.ID, Code: SecondSeasonCode,
		Name: "Battambang crushing season 2026-2027", StartDate: "2026-12-15",
		PlannedDays: 96, Status: "OPEN",
	})
	if err != nil {
		return out, fmt.Errorf("second season: %w", err)
	}
	out.SeasonID = season.ID

	versions, err := s.Planning().ListVersions(ctx, season.ID, store.ListOptions{Top: 100})
	if err != nil {
		return out, err
	}
	var budget domain.PlanVersion
	for _, v := range versions.Items {
		if v.Code == "V1" && v.PlanType != domain.PlanTypeActual {
			budget = v
		}
	}
	if budget.ID == "" {
		budget, err = planning.SaveVersion(ctx, domain.PlanVersion{
			SeasonID: season.ID, Code: "V1", Description: "Battambang season budget",
			PlanType: domain.PlanTypeBudget, Status: domain.StatusDraft,
			EffectiveFrom: season.StartDate,
		})
		if err != nil {
			return out, fmt.Errorf("second budget version: %w", err)
		}
	}
	out.BudgetID = budget.ID

	// Deliberately different figures from Kampong Speu. A second tenant carrying
	// the same numbers would let a scope leak pass unnoticed: the test would be
	// reading the right total from the wrong factory.
	for _, a := range []domain.PlanAssumption{
		{Code: domain.AsmCaneTarget, Description: "Season cane target",
			Value: domain.D("640000"), UOM: "TON"},
		{Code: domain.AsmSeasonDays, Description: "Planned crushing days",
			Value: domain.D("96"), UOM: "DAY"},
		{Code: domain.AsmRecoveryPct, Description: "Raw sugar recovery on cane",
			Value: domain.D("10.40"), UOM: "PCT"},
		{Code: domain.AsmCrushRateTPH, Description: "Crushing rate",
			Value: domain.D("320"), UOM: "TPH"},
		{Code: domain.AsmAvailableHours, Description: "Available crushing hours per day",
			Value: domain.D("22"), UOM: "HOUR"},
		{Code: domain.AsmDirectToRefinePct, Description: "Raw sugar sent straight to refining",
			Value: domain.D("21.875"), UOM: "PCT"},
		{Code: domain.AsmRemeltInputFactor, Description: "Raw sugar per ton of refined output",
			Value: domain.D("1.05"), UOM: "RATIO"},
		{Code: domain.AsmQuotaShipmentTPD, Description: "Planned quota shipment",
			Value: domain.D("180"), UOM: "TPD"},
		{Code: domain.AsmCapacityWarnPct, Description: "Capacity warning threshold",
			Value: domain.D("85"), UOM: "PCT"},
		{Code: domain.AsmCapacityAlertPct, Description: "Capacity alert threshold",
			Value: domain.D("95"), UOM: "PCT"},
		{Code: domain.AsmMassBalanceTolPct, Description: "Mass balance tolerance",
			Value: domain.D("0.5"), UOM: "PCT"},
		{Code: domain.AsmRecoveryMinPct, Description: "Recovery lower bound",
			Value: domain.D("9.2"), UOM: "PCT"},
		{Code: domain.AsmRecoveryMaxPct, Description: "Recovery upper bound",
			Value: domain.D("12.5"), UOM: "PCT"},
	} {
		a.VersionID = budget.ID
		if _, err := planning.SaveAssumption(ctx, a); err != nil {
			return out, fmt.Errorf("second tenant assumption %s: %w", a.Code, err)
		}
	}

	products, err := productIndexByCode(ctx, s)
	if err != nil {
		return out, err
	}
	packaging, err := packagingIndexByCode(ctx, s)
	if err != nil {
		return out, err
	}
	for _, m := range []domain.ProductMixEntry{
		{ProductID: products["WHT"], PackagingID: packaging["P50KG"],
			WarehouseID: out.Warehouses["BTB-FG"], SeasonTons: domain.D("48000")},
		{ProductID: products["REF"], PackagingID: packaging["P50KG"],
			WarehouseID: out.Warehouses["BTB-FG"], SeasonTons: domain.D("14000")},
	} {
		m.VersionID = budget.ID
		if _, err := planning.SaveMixEntry(ctx, m); err != nil {
			return out, fmt.Errorf("second tenant product mix: %w", err)
		}
	}

	generated, err := planning.Generate(ctx, budget.ID, service.GenerateRequest{Replace: true})
	if err != nil {
		return out, fmt.Errorf("generate the second tenant's plan: %w", err)
	}
	out.Generated = generated
	return out, nil
}

func productIndexByCode(ctx context.Context, s store.Store) (map[string]string, error) {
	page, err := s.MasterData().Products().List(ctx, store.ListOptions{Top: 200})
	if err != nil {
		return nil, fmt.Errorf("read the products: %w", err)
	}
	out := map[string]string{}
	for _, p := range page.Items {
		out[p.Code] = p.ID
	}
	return out, nil
}

func packagingIndexByCode(ctx context.Context, s store.Store) (map[string]string, error) {
	page, err := s.MasterData().PackagingTypes().List(ctx, store.ListOptions{Top: 200})
	if err != nil {
		return nil, fmt.Errorf("read the packaging types: %w", err)
	}
	out := map[string]string{}
	for _, p := range page.Items {
		out[p.Code] = p.ID
	}
	return out, nil
}
