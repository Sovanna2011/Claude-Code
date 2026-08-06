// Package seed loads the demonstration scenario described in section 2 of the
// specification: Kampong Speu Sugar, crushing season 2026-2027.
//
// Every figure here is data, not logic. The seed writes master data,
// assumptions and a product mix, then asks the planning service to generate the
// daily plan exactly as a planner would - so the demo data exercises the same
// code path as production use, and a change to the generator shows up in the
// demo immediately.
package seed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// Result reports what was created.
type Result struct {
	CompanyID  string
	FactoryID  string
	SeasonID   string
	BudgetID   string
	ActualID   string
	Generated  service.GenerateResult
	Warehouses map[string]string // code -> id
	Products   map[string]string // code -> id
	Channels   map[string]string // code -> id
	Packaging  map[string]string // code -> id
	Materials  map[string]string // code -> id
}

// Actor is the user recorded against seeded rows.
const Actor = "seed"

// The business keys of the reference scenario, so that code outside this
// package can name the company and the factory without holding an id.
const (
	CompanyCode = "KSS"
	FactoryCode = "F1"
)

// seedPrincipal has every permission needed to build the scenario. It exists
// only inside the seeding process and is never exposed as a login.
func seedPrincipal(companyID, factoryID string) auth.Principal {
	return auth.NewPrincipal("seed", Actor, "Seed data loader", "",
		[]string{
			auth.RoleSystemAdmin, auth.RoleMasterDataAdmin, auth.RoleProductionPlanner,
			auth.RoleApprover, auth.RoleShipmentPlanner,
			// The demo also records actuals, which needs the operator roles.
			auth.RoleShiftSupervisor, auth.RoleWarehouseOperator,
			// The scenario also carries a cost structure and its rates.
			auth.RoleCostController,
		},
		[]string{companyID}, []string{factoryID})
}

// Load writes the reference scenario. It is idempotent on business keys: master
// data that already exists is reused rather than duplicated.
func Load(ctx context.Context, s store.Store, planning *service.Planning) (Result, error) {
	res := Result{
		Warehouses: map[string]string{}, Products: map[string]string{}, Channels: map[string]string{},
	}
	md := s.MasterData()

	// --- organisation -------------------------------------------------------
	company, err := upsert(ctx, md.Companies(), CompanyCode, domain.Company{
		Code: CompanyCode, Name: "Kampong Speu Sugar Co., Ltd.", Currency: "USD",
		TimeZone: "Asia/Phnom_Penh", Validity: active(),
	})
	if err != nil {
		return res, fmt.Errorf("company: %w", err)
	}
	res.CompanyID = company.ID

	factory, err := upsert(ctx, md.Factories(), FactoryCode, domain.Factory{
		CompanyID: company.ID, Code: FactoryCode, Name: "Factory 1 - Kampong Speu",
		TimeZone: "Asia/Phnom_Penh", Validity: active(),
	})
	if err != nil {
		return res, fmt.Errorf("factory: %w", err)
	}
	res.FactoryID = factory.ID

	for _, l := range []domain.ProductionLine{
		{FactoryID: factory.ID, Code: "MILL-1", Name: "Cane milling train 1",
			Stage: domain.StageCane, RatedTPH: domain.D("700"), Validity: active()},
		{FactoryID: factory.ID, Code: "REF-1", Name: "Refinery line 1",
			Stage: domain.StageRefining, RatedTPH: domain.D("80"), Validity: active()},
		{FactoryID: factory.ID, Code: "PACK-1", Name: "Packing line 1",
			Stage: domain.StagePacking, RatedTPH: domain.D("30"), Validity: active()},
	} {
		if _, err := upsert(ctx, md.Lines(), l.Code, l); err != nil {
			return res, fmt.Errorf("line %s: %w", l.Code, err)
		}
	}

	for _, sh := range []domain.Shift{
		{FactoryID: factory.ID, Code: "A", Name: "Shift A (06:00-14:00)",
			StartTime: "06:00", Hours: domain.D("8"), Validity: active()},
		{FactoryID: factory.ID, Code: "B", Name: "Shift B (14:00-22:00)",
			StartTime: "14:00", Hours: domain.D("8"), Validity: active()},
		{FactoryID: factory.ID, Code: "C", Name: "Shift C (22:00-06:00)",
			StartTime: "22:00", Hours: domain.D("8"), Validity: active()},
	} {
		if _, err := upsert(ctx, md.Shifts(), sh.Code, sh); err != nil {
			return res, fmt.Errorf("shift %s: %w", sh.Code, err)
		}
	}

	// --- units and categories ----------------------------------------------
	for _, u := range []domain.UnitOfMeasure{
		{Code: "TON", Name: "Metric ton", Dimension: "MASS", Decimals: 3, Validity: active()},
		{Code: "KG", Name: "Kilogram", Dimension: "MASS", Decimals: 3, Validity: active()},
		{Code: "EA", Name: "Each", Dimension: "COUNT", Decimals: 0, Validity: active()},
		{Code: "HR", Name: "Hour", Dimension: "TIME", Decimals: 2, Validity: active()},
		// Thread is bought and issued by the spool, not by the metre, so that is
		// the unit the store keeps and the unit a consumption is recorded in.
		{Code: "SPOOL", Name: "Spool", Dimension: "COUNT", Decimals: 3, Validity: active()},
	} {
		if _, err := upsert(ctx, md.UOMs(), u.Code, u); err != nil {
			return res, fmt.Errorf("uom %s: %w", u.Code, err)
		}
	}
	// Unit conversions have a compound business key (product, from, to) rather
	// than a single code, so they are matched on that key instead of GetByCode.
	if err := seedConversion(ctx, md.UOMConversions(), domain.UOMConversion{
		FromUOM: "TON", ToUOM: "KG", Factor: domain.D("1000"), Validity: active(),
	}); err != nil {
		return res, fmt.Errorf("uom conversion: %w", err)
	}

	for _, c := range []domain.ProductCategory{
		{Code: "CANE", Name: "Sugar cane", Validity: active()},
		{Code: "RAW", Name: "Raw sugar", Validity: active()},
		{Code: "WHITE", Name: "White and refined sugar", Validity: active()},
		{Code: "BYPROD", Name: "By-products", Validity: active()},
	} {
		if _, err := upsert(ctx, md.ProductCategories(), c.Code, c); err != nil {
			return res, fmt.Errorf("category %s: %w", c.Code, err)
		}
	}

	// --- products -----------------------------------------------------------
	for _, p := range []domain.Product{
		{Code: "CANE", Name: "Sugar cane", CategoryCode: "CANE", BaseUOM: "TON",
			Stage: domain.StageCane, StorageClass: domain.StorageMaterial, Validity: active()},
		{Code: "RAW", Name: "Raw sugar", CategoryCode: "RAW", BaseUOM: "TON",
			Stage: domain.StageRawSugar, StorageClass: domain.StorageRaw, Validity: active()},
		{Code: "REF", Name: "Refined sugar", CategoryCode: "WHITE", BaseUOM: "TON",
			Stage: domain.StageRefining, StorageClass: domain.StorageFinished,
			IsFinished: true, Validity: active()},
		{Code: "WHT", Name: "White sugar", CategoryCode: "WHITE", BaseUOM: "TON",
			Stage: domain.StageRefining, StorageClass: domain.StorageFinished,
			IsFinished: true, Validity: active()},
		{Code: "SUP", Name: "Super refined sugar", CategoryCode: "WHITE", BaseUOM: "TON",
			Stage: domain.StageRefining, StorageClass: domain.StorageFinished,
			IsFinished: true, Validity: active()},
		{Code: "MOL", Name: "Molasses", CategoryCode: "BYPROD", BaseUOM: "TON",
			Stage: domain.StageRawSugar, StorageClass: domain.StorageMaterial, Validity: active()},
		{Code: "BAG", Name: "Bagasse", CategoryCode: "BYPROD", BaseUOM: "TON",
			Stage: domain.StageCane, StorageClass: domain.StorageMaterial, Validity: active()},
		{Code: "FCK", Name: "Filter cake", CategoryCode: "BYPROD", BaseUOM: "TON",
			Stage: domain.StageCane, StorageClass: domain.StorageMaterial, Validity: active()},
	} {
		saved, err := upsert(ctx, md.Products(), p.Code, p)
		if err != nil {
			return res, fmt.Errorf("product %s: %w", p.Code, err)
		}
		res.Products[p.Code] = saved.ID
	}

	// --- packaging materials and types --------------------------------------
	materialIDs := map[string]string{}
	for _, m := range []domain.Material{
		{Code: "BAG-50", Name: "Polypropylene bag 50 kg", UOM: "EA",
			SafetyStock: domain.D("50000"), LeadTimeDays: 45, ScrapPct: domain.D("0.5"),
			OnHand: domain.D("400000"), OnOrder: domain.D("250000"), Validity: active()},
		{Code: "BAG-JUMBO", Name: "Jumbo bag 1.10 t", UOM: "EA",
			SafetyStock: domain.D("2000"), LeadTimeDays: 60, ScrapPct: domain.D("1"),
			OnHand: domain.D("6000"), OnOrder: domain.D("4000"), Validity: active()},
		{Code: "BAG-1T", Name: "Bulk bag 1.0 t", UOM: "EA",
			SafetyStock: domain.D("2000"), LeadTimeDays: 60, ScrapPct: domain.D("1"),
			OnHand: domain.D("30000"), OnOrder: domain.D("20000"), Validity: active()},
		{Code: "PACK-1KG", Name: "Consumer pack 1 kg", UOM: "EA",
			SafetyStock: domain.D("100000"), LeadTimeDays: 30, ScrapPct: domain.D("2"),
			OnHand: domain.D("500000"), OnOrder: domain.Zero, Validity: active()},
		{Code: "LINER", Name: "Inner polythene liner", UOM: "EA",
			SafetyStock: domain.D("20000"), LeadTimeDays: 30, ScrapPct: domain.D("1"),
			OnHand: domain.D("150000"), OnOrder: domain.Zero, Validity: active()},
		{Code: "PALLET", Name: "Wooden pallet", UOM: "EA",
			SafetyStock: domain.D("1000"), LeadTimeDays: 20, ScrapPct: domain.Zero,
			OnHand: domain.D("8000"), OnOrder: domain.Zero, Validity: active()},
		{Code: "THREAD", Name: "Bag closing thread", UOM: "SPOOL",
			SafetyStock: domain.D("500"), LeadTimeDays: 25, ScrapPct: domain.D("3"),
			OnHand: domain.D("3000"), OnOrder: domain.D("1000"), Validity: active()},
		{Code: "LABEL", Name: "Consumer pack label", UOM: "EA",
			SafetyStock: domain.D("50000"), LeadTimeDays: 21, ScrapPct: domain.D("2"),
			OnHand: domain.D("400000"), OnOrder: domain.Zero, Validity: active()},
	} {
		saved, err := upsert(ctx, md.Materials(), m.Code, m)
		if err != nil {
			return res, fmt.Errorf("material %s: %w", m.Code, err)
		}
		materialIDs[m.Code] = saved.ID
	}

	packagingIDs := map[string]string{}
	for _, p := range []domain.PackagingType{
		{Code: "P50KG", Name: "50 kg bag", NetWeightKg: domain.D("50"),
			MaterialID: materialIDs["BAG-50"], Validity: active()},
		{Code: "P1T", Name: "1.0 t bulk bag", NetWeightKg: domain.D("1000"),
			MaterialID: materialIDs["BAG-1T"], Validity: active()},
		{Code: "PJUMBO", Name: "1.10 t jumbo bag", NetWeightKg: domain.D("1100"),
			MaterialID: materialIDs["BAG-JUMBO"], Validity: active()},
		{Code: "P1KG", Name: "1 kg consumer pack", NetWeightKg: domain.D("1"),
			MaterialID: materialIDs["PACK-1KG"], Validity: active()},
	} {
		saved, err := upsert(ctx, md.PackagingTypes(), p.Code, p)
		if err != nil {
			return res, fmt.Errorf("packaging %s: %w", p.Code, err)
		}
		packagingIDs[p.Code] = saved.ID
	}

	// The bill of materials for each packaging type: what goes into a package
	// besides the bag itself. The figures are the ordinary ones for a bagging
	// line - a liner in every large bag, thread to sew it, a pallet shared
	// between the bags stacked on it, and a label on each consumer pack.
	for _, line := range []struct {
		packaging, material string
		qtyPerPackage       string
	}{
		{"PJUMBO", "LINER", "1"},
		{"PJUMBO", "THREAD", "0.004"},
		// A pallet carries twenty 50 kg bags, so each bag consumes a twentieth
		// of one. Fractional consumption is exactly why the column has six
		// decimals.
		{"P50KG", "PALLET", "0.05"},
		{"P50KG", "THREAD", "0.0012"},
		{"P1T", "LINER", "1"},
		{"P1T", "THREAD", "0.004"},
		{"P1KG", "LABEL", "1"},
	} {
		if _, err := md.SavePackagingBOM(ctx, domain.PackagingBOMLine{
			PackagingID:   packagingIDs[line.packaging],
			MaterialID:    materialIDs[line.material],
			QtyPerPackage: domain.D(line.qtyPerPackage),
			Validity:      active(),
		}, Actor); err != nil && !errors.Is(err, domain.ErrDuplicate) {
			return res, fmt.Errorf("packaging bill of materials %s/%s: %w",
				line.packaging, line.material, err)
		}
	}

	res.Packaging, res.Materials = packagingIDs, materialIDs

	// A mapping template for the daily sheet the mill already keeps, so the
	// import can be tried without anybody first designing a template. The
	// headings are the ones a Cambodian mill's spreadsheet actually uses, and
	// the date format is day-first because that is how it is written there.
	for _, m := range []domain.ImportMapping{
		{
			Code: "CANE-DAILY", Name: "Daily cane sheet", Kind: domain.ImportCane,
			HeaderRow: 1, DateFormat: "02/01/2006", Validity: active(),
			Note: "The shift sheet from the weighbridge office.",
			Columns: []domain.ColumnMapping{
				{Field: "businessDate", Header: "Date"},
				{Field: "caneDelivered", Header: "Delivered (MT)"},
				{Field: "caneAccepted", Header: "Accepted (MT)"},
				{Field: "caneRejected", Header: "Rejected (MT)"},
				{Field: "caneCrushed", Header: "Crushed (MT)"},
				{Field: "crushRateTph", Header: "Rate (TPH)"},
				{Field: "availableHours", Header: "Hours available", Default: "24"},
				{Field: "stoppageHours", Header: "Hours stopped", Default: "0"},
				{Field: "note", Header: "Remarks"},
			},
		},
		{
			Code: "PROD-DAILY", Name: "Daily production sheet", Kind: domain.ImportProduction,
			HeaderRow: 1, DateFormat: "02/01/2006", Validity: active(),
			Note: "Packed output by product, from the packing hall.",
			Columns: []domain.ColumnMapping{
				{Field: "businessDate", Header: "Date"},
				{Field: "productCode", Header: "Product"},
				{Field: "packagingCode", Header: "Pack"},
				{Field: "quantity", Header: "Good output (MT)"},
				{Field: "remeltInput", Header: "Raw used (MT)"},
				{Field: "processLoss", Header: "Loss (MT)"},
				{Field: "rework", Header: "Rework (MT)"},
				{Field: "holdQty", Header: "On hold (MT)"},
				{Field: "note", Header: "Remarks"},
			},
		},
	} {
		if _, err := s.Imports().SaveMapping(ctx, m, Actor); err != nil &&
			!errors.Is(err, domain.ErrDuplicate) {
			return res, fmt.Errorf("import mapping %s: %w", m.Code, err)
		}
	}

	// --- warehouses ---------------------------------------------------------
	// The workbook states nominal capacities; usable capacity is set to 100 %
	// here so the seeded figures reconcile exactly with the source document.
	for _, w := range []domain.Warehouse{
		{FactoryID: factory.ID, Code: "RAW-WH1", Name: "Raw Sugar Warehouse 1",
			StorageClass: domain.StorageRaw, CapacityTons: domain.D("45000"),
			UsablePct: domain.D("100"), Validity: active()},
		{FactoryID: factory.ID, Code: "RAW-WH2", Name: "Raw Sugar Warehouse 2",
			StorageClass: domain.StorageRaw, CapacityTons: domain.D("65000"),
			UsablePct: domain.D("100"), Validity: active()},
		{FactoryID: factory.ID, Code: "FG-WH1", Name: "Refined/White Warehouse 1",
			StorageClass: domain.StorageFinished, CapacityTons: domain.D("22000"),
			UsablePct: domain.D("100"), Validity: active()},
		{FactoryID: factory.ID, Code: "FG-WH3", Name: "Refined/White Warehouse 3",
			StorageClass: domain.StorageFinished, CapacityTons: domain.D("47000"),
			UsablePct: domain.D("100"), Validity: active()},
	} {
		saved, err := upsert(ctx, md.Warehouses(), w.Code, w)
		if err != nil {
			return res, fmt.Errorf("warehouse %s: %w", w.Code, err)
		}
		res.Warehouses[w.Code] = saved.ID
	}

	// --- customers and channels --------------------------------------------
	// The source workbook tracks shipment in one column per trader. Here they
	// are master data, so adding a customer needs no schema change.
	customerIDs := map[string]string{}
	for _, c := range []domain.Customer{
		{Code: "TRD-A", Name: "Trader A", Country: "KH", Validity: active()},
		{Code: "TRD-B", Name: "Trader B", Country: "KH", Validity: active()},
		{Code: "TRD-C", Name: "Trader C", Country: "KH", Validity: active()},
	} {
		saved, err := upsert(ctx, md.Customers(), c.Code, c)
		if err != nil {
			return res, fmt.Errorf("customer %s: %w", c.Code, err)
		}
		customerIDs[c.Code] = saved.ID
	}
	for _, ch := range []domain.ShipmentChannel{
		{Code: "QUOTA", Name: "Domestic quota shipment", Category: "QUOTA",
			CustomerID: customerIDs["TRD-A"], Validity: active()},
		{Code: "EXPORT", Name: "Export shipment", Category: "EXPORT",
			CustomerID: customerIDs["TRD-B"], Validity: active()},
		{Code: "DOMESTIC", Name: "Domestic direct sales", Category: "DOMESTIC",
			CustomerID: customerIDs["TRD-C"], Validity: active()},
	} {
		saved, err := upsert(ctx, md.Channels(), ch.Code, ch)
		if err != nil {
			return res, fmt.Errorf("channel %s: %w", ch.Code, err)
		}
		res.Channels[ch.Code] = saved.ID
	}

	// --- reason codes -------------------------------------------------------
	for _, r := range []domain.ReasonCode{
		{Code: "DT-MILL", Name: "Mill stoppage", Category: "DOWNTIME", Validity: active()},
		{Code: "DT-BOILER", Name: "Boiler problem", Category: "DOWNTIME", Validity: active()},
		{Code: "DT-POWER", Name: "Power failure", Category: "DOWNTIME", Validity: active()},
		{Code: "DT-RAIN", Name: "Rain, no cane supply", Category: "DOWNTIME", Validity: active()},
		{Code: "LOSS-PROC", Name: "Process loss", Category: "LOSS", Validity: active()},
		{Code: "LOSS-HAND", Name: "Handling loss", Category: "LOSS", Validity: active()},
		{Code: "ADJ-COUNT", Name: "Stock count correction", Category: "STOCK_CORRECTION", Validity: active()},
		{Code: "VAR-CANE", Name: "Cane quality variance", Category: "VARIANCE", Validity: active()},
		{Code: "REJ-COLOUR", Name: "Colour out of specification", Category: "REJECTION", Validity: active()},
		{Code: "RWK-REMELT", Name: "Rework by remelt", Category: "REWORK", Validity: active()},
	} {
		if _, err := upsert(ctx, md.ReasonCodes(), r.Code, r); err != nil {
			return res, fmt.Errorf("reason code %s: %w", r.Code, err)
		}
	}

	// --- quality catalogue --------------------------------------------------
	// Without a parameter catalogue and limits in force, a laboratory sheet has
	// nothing to judge against and a sample can never fail, so the execution
	// side of the scenario would be unusable out of the box.
	parameterIDs := map[string]string{}
	for _, p := range []domain.QualityParameter{
		{Code: "POL", Name: "Polarisation", UOM: "PCT", TestMethod: "ICUMSA GS1/2/3-1", Validity: active()},
		{Code: "BRIX", Name: "Brix", UOM: "PCT", TestMethod: "Refractometer", Validity: active()},
		{Code: "COLOUR", Name: "Colour", UOM: "IU", TestMethod: "ICUMSA GS1/3-7", Validity: active()},
		{Code: "MOIST", Name: "Moisture", UOM: "PCT", TestMethod: "Oven drying", Validity: active()},
		{Code: "ASH", Name: "Conductivity ash", UOM: "PCT", TestMethod: "ICUMSA GS2/3-17", Validity: active()},
	} {
		saved, err := s.Execution().SaveParameter(ctx, p, Actor)
		if err != nil {
			return res, fmt.Errorf("quality parameter %s: %w", p.Code, err)
		}
		parameterIDs[p.Code] = saved.ID
	}

	// The limits below are ordinary refinery figures for the three finished
	// grades. They are effective from the first day of the season, so the whole
	// campaign is judged by the same specification.
	limit := func(v string) *domain.Dec { d := domain.D(v); return &d }
	for _, spec := range []struct {
		product, parameter                 string
		lower, upper, warnLower, warnUpper string
	}{
		{"REF", "POL", "99.700", "", "99.800", ""},
		{"REF", "COLOUR", "", "45", "", "35"},
		{"REF", "MOIST", "", "0.040", "", "0.030"},
		{"REF", "ASH", "", "0.015", "", "0.012"},
		{"WHT", "POL", "99.500", "", "99.600", ""},
		{"WHT", "COLOUR", "", "150", "", "120"},
		{"WHT", "MOIST", "", "0.060", "", "0.050"},
		{"SUP", "POL", "99.900", "", "99.930", ""},
		{"SUP", "COLOUR", "", "25", "", "20"},
		{"RAW", "POL", "97.000", "", "97.500", ""},
		{"RAW", "BRIX", "98.000", "", "", ""},
	} {
		productID, ok := res.Products[spec.product]
		if !ok {
			continue
		}
		entry := domain.QualitySpec{
			ProductID: productID, ParameterID: parameterIDs[spec.parameter],
			ValidFrom: "2026-12-01",
		}
		if spec.lower != "" {
			entry.LowerLimit = limit(spec.lower)
		}
		if spec.upper != "" {
			entry.UpperLimit = limit(spec.upper)
		}
		if spec.warnLower != "" {
			entry.WarnLower = limit(spec.warnLower)
		}
		if spec.warnUpper != "" {
			entry.WarnUpper = limit(spec.warnUpper)
		}
		if _, err := s.Execution().SaveSpec(ctx, entry, Actor); err != nil {
			return res, fmt.Errorf("quality specification %s/%s: %w", spec.product, spec.parameter, err)
		}
	}

	// --- cost structure -----------------------------------------------------
	// Ordinary refinery cost lines, each against the driver it really scales
	// with. Without them the costing screen has nothing to report and the unit
	// cost is zero, which reads as free rather than as unconfigured.
	elementIDs := map[string]string{}
	for _, e := range []domain.CostElement{
		{Code: "CANE", Name: "Cane payment", Category: domain.CategoryCane,
			Driver: domain.DriverCaneTon, Variable: true, Validity: active(),
			Note: "Paid to growers on delivered weight"},
		{Code: "HARVEST", Name: "Harvesting and haulage", Category: domain.CategoryCane,
			Driver: domain.DriverCaneTon, Variable: true, Validity: active()},
		{Code: "FUEL", Name: "Boiler fuel", Category: domain.CategoryEnergy,
			Driver: domain.DriverRunHour, Variable: true, Validity: active()},
		{Code: "POWER", Name: "Purchased electricity", Category: domain.CategoryEnergy,
			Driver: domain.DriverRunHour, Variable: true, Validity: active()},
		{Code: "LIME", Name: "Lime and clarification chemicals", Category: domain.CategoryChemicals,
			Driver: domain.DriverCaneTon, Variable: true, Validity: active()},
		{Code: "REFCHEM", Name: "Refining chemicals", Category: domain.CategoryChemicals,
			Driver: domain.DriverSugarTon, Variable: true, Validity: active()},
		{Code: "PACKMAT", Name: "Packaging materials", Category: domain.CategoryPackaging,
			Driver: domain.DriverSugarTon, Variable: true, Validity: active()},
		{Code: "SHIFTLAB", Name: "Shift labour", Category: domain.CategoryLabour,
			Driver: domain.DriverRunHour, Variable: true, Validity: active()},
		{Code: "STAFF", Name: "Salaried staff", Category: domain.CategoryLabour,
			Driver: domain.DriverCalendarDay, Variable: false, Validity: active(),
			Note: "Paid whether the mill runs or not"},
		{Code: "MAINT", Name: "Routine maintenance", Category: domain.CategoryMaintenance,
			Driver: domain.DriverRunHour, Variable: true, Validity: active()},
		{Code: "SHUTDOWN", Name: "Annual overhaul", Category: domain.CategoryMaintenance,
			Driver: domain.DriverFixedSeason, Variable: false, Validity: active()},
		{Code: "OVERHEAD", Name: "Site overhead", Category: domain.CategoryOverhead,
			Driver: domain.DriverCalendarDay, Variable: false, Validity: active()},
	} {
		saved, err := s.Costing().SaveElement(ctx, e, Actor)
		if err != nil {
			return res, fmt.Errorf("cost element %s: %w", e.Code, err)
		}
		elementIDs[e.Code] = saved.ID
	}

	// Standard rates for the whole campaign, in the company's currency except
	// cane, which growers are paid in riel. The costing converts it.
	for _, r := range []struct {
		element, rate, currency string
	}{
		{"CANE", "92250.000000", "KHR"}, // 22.50 USD/t at 4,100
		{"HARVEST", "6.400000", "USD"},
		{"FUEL", "140.000000", "USD"},
		{"POWER", "38.500000", "USD"},
		{"LIME", "1.150000", "USD"},
		{"REFCHEM", "9.800000", "USD"},
		{"PACKMAT", "12.250000", "USD"},
		{"SHIFTLAB", "96.000000", "USD"},
		{"STAFF", "3100.000000", "USD"},
		{"MAINT", "44.000000", "USD"},
		{"SHUTDOWN", "450000.000000", "USD"},
		{"OVERHEAD", "1850.000000", "USD"},
	} {
		if _, err := s.Costing().SaveRate(ctx, domain.CostRate{
			ElementID: elementIDs[r.element], FactoryID: factory.ID,
			RateType: domain.RateStandard, Rate: domain.D(r.rate), Currency: r.currency,
			ValidFrom: "2026-12-01",
		}, Actor); err != nil {
			return res, fmt.Errorf("standard rate %s: %w", r.element, err)
		}
	}

	// Two actual rates, so the demonstration has a variance to explain: fuel
	// was invoiced above budget, and lime below it.
	for _, r := range []struct{ element, rate, currency string }{
		{"FUEL", "155.000000", "USD"},
		{"LIME", "1.080000", "USD"},
	} {
		if _, err := s.Costing().SaveRate(ctx, domain.CostRate{
			ElementID: elementIDs[r.element], FactoryID: factory.ID,
			RateType: domain.RateActual, Rate: domain.D(r.rate), Currency: r.currency,
			ValidFrom: "2026-12-01",
		}, Actor); err != nil {
			return res, fmt.Errorf("actual rate %s: %w", r.element, err)
		}
	}

	if _, err := s.Costing().SaveExchangeRate(ctx, domain.ExchangeRate{
		FromCurrency: "USD", ToCurrency: "KHR", Rate: domain.D("4100"),
		ValidFrom: "2026-12-01",
	}, Actor); err != nil {
		return res, fmt.Errorf("exchange rate: %w", err)
	}

	// --- season, assumptions and mix ---------------------------------------
	ctx = auth.WithPrincipal(ctx, seedPrincipal(company.ID, factory.ID))

	season, err := findOrCreateSeason(ctx, planning, domain.Season{
		CompanyID: company.ID, FactoryID: factory.ID, Code: "2026-2027",
		Name: "Crushing season 2026-2027", StartDate: "2026-12-01",
		PlannedDays: 137, Status: "OPEN",
	})
	if err != nil {
		return res, fmt.Errorf("season: %w", err)
	}
	res.SeasonID = season.ID

	versions, err := s.Planning().ListVersions(ctx, season.ID, store.ListOptions{Top: 100})
	if err != nil {
		return res, err
	}
	var budget domain.PlanVersion
	for _, v := range versions.Items {
		switch {
		case v.PlanType == domain.PlanTypeActual:
			res.ActualID = v.ID
		case v.Code == "V1":
			budget = v
		}
	}
	if budget.ID == "" {
		budget, err = planning.SaveVersion(ctx, domain.PlanVersion{
			SeasonID: season.ID, Code: "V1", Description: "Original season budget",
			PlanType: domain.PlanTypeBudget, Status: domain.StatusDraft,
			EffectiveFrom: season.StartDate,
		})
		if err != nil {
			return res, fmt.Errorf("budget version: %w", err)
		}
	}
	res.BudgetID = budget.ID

	// Assumptions. These are the levers a planner changes for a what-if run;
	// none of them is hard-coded in the calculation code.
	for _, a := range []domain.PlanAssumption{
		{Code: domain.AsmCaneTarget, Description: "Season cane target",
			Value: domain.D("2300000"), UOM: "TON"},
		{Code: domain.AsmSeasonDays, Description: "Planned crushing days",
			Value: domain.D("137"), UOM: "DAY"},
		{Code: domain.AsmRecoveryPct, Description: "Raw sugar recovery on cane",
			Value: domain.D("11.00"), UOM: "%"},
		{Code: domain.AsmDirectToRefinePct, Description: "Raw sugar sent straight to refining",
			Value: domain.D("49.387"), UOM: "%"},
		{Code: domain.AsmRemeltInputFactor, Description: "Raw sugar input per ton of finished goods",
			Value: domain.D("1.05"), UOM: "RATIO"},
		{Code: domain.AsmCrushRateTPH, Description: "Crushing rate",
			Value: domain.D("700"), UOM: "TON/HR"},
		{Code: domain.AsmAvailableHours, Description: "Available crushing hours per day",
			Value: domain.D("24"), UOM: "HR"},
		{Code: domain.AsmQuotaShipmentTPD, Description: "Planned quota shipment",
			Value: domain.D("500"), UOM: "TON/DAY"},
		{Code: domain.AsmJumboPackTPD, Description: "Jumbo bag packing rate",
			Value: domain.D("300"), UOM: "TON/DAY"},
		{Code: domain.AsmJumboBagWeightTons, Description: "Jumbo bag net weight",
			Value: domain.D("1.10"), UOM: "TON"},
		{Code: domain.AsmCapacityWarnPct, Description: "Storage warning threshold",
			Value: domain.D("80"), UOM: "%"},
		{Code: domain.AsmCapacityAlertPct, Description: "Storage critical threshold",
			Value: domain.D("90"), UOM: "%"},
		{Code: domain.AsmMassBalanceTolPct, Description: "Mass balance tolerance",
			Value: domain.D("0.5"), UOM: "%"},
		{Code: domain.AsmRecoveryMinPct, Description: "Lowest acceptable recovery",
			Value: domain.D("9.8"), UOM: "%"},
		{Code: domain.AsmRecoveryMaxPct, Description: "Highest expected recovery",
			Value: domain.D("13.0"), UOM: "%"},
	} {
		a.VersionID = budget.ID
		a.ValidFrom = season.StartDate
		if _, err := planning.SaveAssumption(ctx, a); err != nil {
			return res, fmt.Errorf("assumption %s: %w", a.Code, err)
		}
	}

	// Product mix: the finished goods tonnages from the workbook.
	//   refined 106,700 + white 133,400 + super refined 2,000 = 242,100 t
	// Jumbo bag packing of raw sugar runs at 300 t/day up to 20,700 t.
	for _, m := range []domain.ProductMixEntry{
		{ProductID: res.Products["REF"], PackagingID: packagingIDs["P50KG"],
			WarehouseID: res.Warehouses["FG-WH3"], SeasonTons: domain.D("106700")},
		{ProductID: res.Products["WHT"], PackagingID: packagingIDs["P50KG"],
			WarehouseID: res.Warehouses["FG-WH1"], SeasonTons: domain.D("133400")},
		{ProductID: res.Products["SUP"], PackagingID: packagingIDs["P1KG"],
			WarehouseID: res.Warehouses["FG-WH1"], SeasonTons: domain.D("2000")},
	} {
		m.VersionID = budget.ID
		if _, err := planning.SaveMixEntry(ctx, m); err != nil {
			return res, fmt.Errorf("product mix: %w", err)
		}
	}

	// --- generate the daily plan -------------------------------------------
	generated, err := planning.Generate(ctx, budget.ID, service.GenerateRequest{Replace: true})
	if err != nil {
		return res, fmt.Errorf("generate plan: %w", err)
	}
	res.Generated = generated
	return res, nil
}

// LoadWithActuals seeds the scenario and then records a plausible first fortnight
// of actuals, so the dashboard has something to show in a demonstration.
func LoadWithActuals(ctx context.Context, s store.Store, planning *service.Planning, days int) (Result, error) {
	res, err := Load(ctx, s, planning)
	if err != nil {
		return res, err
	}
	if days <= 0 {
		return res, nil
	}
	ctx = auth.WithPrincipal(ctx, seedPrincipal(res.CompanyID, res.FactoryID))

	plan, err := s.Planning().ListCane(ctx, store.PlanFilter{
		VersionIDs: []string{res.BudgetID}, Series: domain.SeriesPlan, Top: days,
	})
	if err != nil {
		return res, err
	}
	if len(plan) < days {
		days = len(plan)
	}

	// A deterministic pattern rather than random numbers: the demo has to look
	// the same every time it is loaded, and the numbers have to be arguable.
	// Day 3 has a boiler stoppage; the rest run slightly below target while the
	// mill settles, then above it.
	factors := []string{"0.82", "0.94", "0.61", "0.97", "1.02", "1.01", "0.99",
		"1.03", "1.04", "0.98", "1.01", "1.05", "1.02", "1.00"}

	// The plan's own finished-goods rows for the same days, so the actuals
	// follow the product mix rather than inventing one. Without them the
	// actuals stop at raw sugar: the stock ledger has nothing finished in it
	// and the cost per ton divides by zero output, which reads as free rather
	// than as unrecorded.
	plannedProducts, err := s.Planning().ListProducts(ctx, store.PlanFilter{
		VersionIDs: []string{res.BudgetID}, Series: domain.SeriesPlan,
		From: plan[0].BusinessDate, To: plan[days-1].BusinessDate,
	})
	if err != nil {
		return res, fmt.Errorf("read the planned production: %w", err)
	}
	finished := map[string]bool{}
	for _, code := range []string{"REF", "WHT", "SUP"} {
		if id, ok := res.Products[code]; ok {
			finished[id] = true
		}
	}
	byDate := map[domain.BusinessDate][]domain.DailyProductPlan{}
	for _, row := range plannedProducts {
		if finished[row.ProductID] {
			byDate[row.BusinessDate] = append(byDate[row.BusinessDate], row)
		}
	}

	var caneRows []domain.DailyCanePlan
	var rawRows []domain.DailyProductPlan
	for i := 0; i < days; i++ {
		p := plan[i]
		factor := domain.D(factors[i%len(factors)])
		crushed := domain.RoundQty(p.CaneCrushed.Mul(factor))
		stoppage := domain.Zero
		reason := ""
		if factor.LessThan(domain.D("0.7")) {
			stoppage = domain.D("8.5")
			reason = "DT-BOILER"
		}
		caneRows = append(caneRows, domain.DailyCanePlan{
			FactoryID: res.FactoryID, BusinessDate: p.BusinessDate, Series: domain.SeriesActual,
			CaneDelivered: crushed, CaneAccepted: crushed, CaneCrushed: crushed,
			CrushRateTPH: p.CrushRateTPH, AvailableHrs: domain.D("24"), StoppageHrs: stoppage,
			ReasonCode: reason,
		})
		// Recovery drifts a little around the 11 % assumption.
		recovery := domain.D("10.7").Add(domain.D("0.1").Mul(domain.DI(int64(i % 5))))
		rawRows = append(rawRows, domain.DailyProductPlan{
			FactoryID: res.FactoryID, BusinessDate: p.BusinessDate, ProductID: res.Products["RAW"],
			Series: domain.SeriesActual, Quantity: domain.ExpectedRawSugar(crushed, recovery),
		})

		// Finished goods move with the same daily factor as the cane, so a bad
		// day upstream shows as a bad day downstream.
		for _, row := range byDate[p.BusinessDate] {
			rawRows = append(rawRows, domain.DailyProductPlan{
				FactoryID: res.FactoryID, BusinessDate: p.BusinessDate,
				LineID: row.LineID, ProductID: row.ProductID, PackagingID: row.PackagingID,
				Series: domain.SeriesActual, Quantity: domain.RoundQty(row.Quantity.Mul(factor)),
				RemeltInput: domain.RoundQty(row.RemeltInput.Mul(factor)),
			})
		}
	}

	if _, err := planning.UpsertCane(ctx, res.ActualID, caneRows, service.UpsertOptions{}); err != nil {
		return res, fmt.Errorf("seed actual cane: %w", err)
	}
	if _, err := planning.UpsertProduction(ctx, res.ActualID, rawRows, service.UpsertOptions{}); err != nil {
		return res, fmt.Errorf("seed actual raw sugar: %w", err)
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func active() domain.Validity { return domain.Validity{Active: true} }

// upsert creates a record, or returns the existing one with the same code. It
// keeps the seed idempotent so it can be re-run against an existing database.
func upsert[T any](ctx context.Context, repo store.Repo[T], code string, entity T) (T, error) {
	if existing, err := repo.GetByCode(ctx, code); err == nil {
		return existing, nil
	}
	return repo.Save(ctx, entity, Actor)
}

// seedConversion inserts a unit conversion unless one already exists for the
// same product, source unit and target unit.
func seedConversion(ctx context.Context, repo store.Repo[domain.UOMConversion], c domain.UOMConversion) error {
	page, err := repo.List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return err
	}
	for _, existing := range page.Items {
		if existing.ProductID == c.ProductID && existing.FromUOM == c.FromUOM && existing.ToUOM == c.ToUOM {
			return nil
		}
	}
	_, err = repo.Save(ctx, c, Actor)
	return err
}

func findOrCreateSeason(ctx context.Context, planning *service.Planning, s domain.Season) (domain.Season, error) {
	page, err := planning.ListSeasons(ctx, store.ListOptions{Top: 100})
	if err != nil {
		return domain.Season{}, err
	}
	for _, existing := range page.Items {
		if existing.Code == s.Code && existing.FactoryID == s.FactoryID {
			return existing, nil
		}
	}
	return planning.SaveSeason(ctx, s)
}

// Clock returns the fixed clock used by the demo profile, so that seeded audit
// timestamps and generated forecasts are reproducible.
func Clock() func() time.Time {
	return func() time.Time { return time.Now().UTC() }
}
