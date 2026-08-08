package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type masterData struct{ s *Store }

// MasterData returns the master-data repositories.
func (s *Store) MasterData() store.MasterData { return masterData{s} }

// scanValidity reads the three effective-dating columns.
type validityCols struct {
	from   *time.Time
	to     *time.Time
	active bool
}

func (v validityCols) toDomain() domain.Validity {
	return domain.Validity{ValidFrom: bd(v.from), ValidTo: bd(v.to), Active: v.active}
}

func (m masterData) Companies() store.Repo[domain.Company] {
	return pgRepo[domain.Company]{s: m.s, sp: pgSpec[domain.Company]{
		name: "company", table: "companies", codeCol: "code",
		cols:      []string{"code", "name", "currency", "time_zone", "valid_from", "valid_to", "active"},
		parentCol: "", textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(c *domain.Company) *string { return &c.ID },
		code:  func(c domain.Company) string { return c.Code },
		audit: func(c *domain.Company) *domain.AuditFields { return &c.AuditFields },
		values: func(c domain.Company) []any {
			return []any{c.Code, c.Name, c.Currency, c.TimeZone, nd(c.ValidFrom), nd(c.ValidTo), c.Active}
		},
		scan: func(r scanner) (domain.Company, error) {
			var c domain.Company
			var v validityCols
			err := r.Scan(&c.ID, &c.Code, &c.Name, &c.Currency, &c.TimeZone,
				&v.from, &v.to, &v.active,
				&c.CreatedAt, &c.CreatedBy, &c.UpdatedAt, &c.UpdatedBy, &c.RowVersion)
			c.Validity = v.toDomain()
			return c, err
		},
	}}
}

func (m masterData) Factories() store.Repo[domain.Factory] {
	return pgRepo[domain.Factory]{s: m.s, sp: pgSpec[domain.Factory]{
		name: "factory", table: "factories", codeCol: "code",
		cols:      []string{"company_id", "code", "name", "time_zone", "valid_from", "valid_to", "active"},
		parentCol: "company_id", textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(f *domain.Factory) *string { return &f.ID },
		code:  func(f domain.Factory) string { return f.Code },
		audit: func(f *domain.Factory) *domain.AuditFields { return &f.AuditFields },
		values: func(f domain.Factory) []any {
			return []any{f.CompanyID, f.Code, f.Name, f.TimeZone, nd(f.ValidFrom), nd(f.ValidTo), f.Active}
		},
		scan: func(r scanner) (domain.Factory, error) {
			var f domain.Factory
			var v validityCols
			err := r.Scan(&f.ID, &f.CompanyID, &f.Code, &f.Name, &f.TimeZone,
				&v.from, &v.to, &v.active,
				&f.CreatedAt, &f.CreatedBy, &f.UpdatedAt, &f.UpdatedBy, &f.RowVersion)
			f.Validity = v.toDomain()
			return f, err
		},
	}}
}

func (m masterData) Lines() store.Repo[domain.ProductionLine] {
	return pgRepo[domain.ProductionLine]{s: m.s, sp: pgSpec[domain.ProductionLine]{
		name: "production line", table: "production_lines", codeCol: "code",
		cols:      []string{"factory_id", "code", "name", "stage", "rated_tph", "valid_from", "valid_to", "active"},
		parentCol: "factory_id", textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(l *domain.ProductionLine) *string { return &l.ID },
		code:  func(l domain.ProductionLine) string { return l.Code },
		audit: func(l *domain.ProductionLine) *domain.AuditFields { return &l.AuditFields },
		values: func(l domain.ProductionLine) []any {
			return []any{l.FactoryID, l.Code, l.Name, string(l.Stage), l.RatedTPH,
				nd(l.ValidFrom), nd(l.ValidTo), l.Active}
		},
		scan: func(r scanner) (domain.ProductionLine, error) {
			var l domain.ProductionLine
			var v validityCols
			var stage string
			err := r.Scan(&l.ID, &l.FactoryID, &l.Code, &l.Name, &stage, &l.RatedTPH,
				&v.from, &v.to, &v.active,
				&l.CreatedAt, &l.CreatedBy, &l.UpdatedAt, &l.UpdatedBy, &l.RowVersion)
			l.Stage, l.Validity = domain.ProcessStage(stage), v.toDomain()
			return l, err
		},
	}}
}

func (m masterData) Shifts() store.Repo[domain.Shift] {
	return pgRepo[domain.Shift]{s: m.s, sp: pgSpec[domain.Shift]{
		name: "shift", table: "shifts", codeCol: "code",
		cols:      []string{"factory_id", "code", "name", "start_time", "hours", "valid_from", "valid_to", "active"},
		parentCol: "factory_id", textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(x *domain.Shift) *string { return &x.ID },
		code:  func(x domain.Shift) string { return x.Code },
		audit: func(x *domain.Shift) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.Shift) []any {
			start := x.StartTime
			if start == "" {
				start = "00:00"
			}
			return []any{x.FactoryID, x.Code, x.Name, start, x.Hours, nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.Shift, error) {
			var x domain.Shift
			var v validityCols
			var start time.Time
			err := r.Scan(&x.ID, &x.FactoryID, &x.Code, &x.Name, &start, &x.Hours,
				&v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.StartTime, x.Validity = start.Format("15:04"), v.toDomain()
			return x, err
		},
	}}
}

func (m masterData) ProductCategories() store.Repo[domain.ProductCategory] {
	return pgRepo[domain.ProductCategory]{s: m.s, sp: pgSpec[domain.ProductCategory]{
		name: "product category", table: "product_categories", codeCol: "code",
		cols:     []string{"code", "name", "valid_from", "valid_to", "active"},
		textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(x *domain.ProductCategory) *string { return &x.ID },
		code:  func(x domain.ProductCategory) string { return x.Code },
		audit: func(x *domain.ProductCategory) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.ProductCategory) []any {
			return []any{x.Code, x.Name, nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.ProductCategory, error) {
			var x domain.ProductCategory
			var v validityCols
			err := r.Scan(&x.ID, &x.Code, &x.Name, &v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.Validity = v.toDomain()
			return x, err
		},
	}}
}

func (m masterData) Products() store.Repo[domain.Product] {
	return pgRepo[domain.Product]{s: m.s, sp: pgSpec[domain.Product]{
		name: "product", table: "products", codeCol: "code",
		cols: []string{"code", "name", "category_code", "base_uom", "stage", "storage_class",
			"is_finished", "valid_from", "valid_to", "active"},
		parentCol: "category_code", textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(x *domain.Product) *string { return &x.ID },
		code:  func(x domain.Product) string { return x.Code },
		audit: func(x *domain.Product) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.Product) []any {
			return []any{x.Code, x.Name, x.CategoryCode, x.BaseUOM, string(x.Stage),
				string(x.StorageClass), x.IsFinished, nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.Product, error) {
			var x domain.Product
			var v validityCols
			var stage, class string
			err := r.Scan(&x.ID, &x.Code, &x.Name, &x.CategoryCode, &x.BaseUOM, &stage, &class,
				&x.IsFinished, &v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.Stage, x.StorageClass, x.Validity = domain.ProcessStage(stage), domain.StorageClass(class), v.toDomain()
			return x, err
		},
	}}
}

func (m masterData) UOMs() store.Repo[domain.UnitOfMeasure] {
	return pgRepo[domain.UnitOfMeasure]{s: m.s, sp: pgSpec[domain.UnitOfMeasure]{
		name: "unit of measure", table: "units_of_measure", codeCol: "code",
		cols:     []string{"code", "name", "dimension", "decimals", "valid_from", "valid_to", "active"},
		textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(x *domain.UnitOfMeasure) *string { return &x.ID },
		code:  func(x domain.UnitOfMeasure) string { return x.Code },
		audit: func(x *domain.UnitOfMeasure) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.UnitOfMeasure) []any {
			return []any{x.Code, x.Name, x.Dimension, x.Decimals, nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.UnitOfMeasure, error) {
			var x domain.UnitOfMeasure
			var v validityCols
			err := r.Scan(&x.ID, &x.Code, &x.Name, &x.Dimension, &x.Decimals,
				&v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.Validity = v.toDomain()
			return x, err
		},
	}}
}

func (m masterData) UOMConversions() store.Repo[domain.UOMConversion] {
	return pgRepo[domain.UOMConversion]{s: m.s, sp: pgSpec[domain.UOMConversion]{
		name: "unit conversion", table: "uom_conversions", codeCol: "from_uom",
		cols:      []string{"product_id", "from_uom", "to_uom", "factor", "valid_from", "valid_to", "active"},
		parentCol: "product_id", textCols: []string{"from_uom", "to_uom"}, activeCol: "active",
		id: func(x *domain.UOMConversion) *string { return &x.ID },
		// The business key is compound; the same string is used by the in-memory
		// store so an error message reads the same whichever store is running.
		code:  func(x domain.UOMConversion) string { return x.ProductID + "/" + x.FromUOM + "/" + x.ToUOM },
		audit: func(x *domain.UOMConversion) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.UOMConversion) []any {
			return []any{nu(x.ProductID), x.FromUOM, x.ToUOM, x.Factor, nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.UOMConversion, error) {
			var x domain.UOMConversion
			var v validityCols
			var product *string
			err := r.Scan(&x.ID, &product, &x.FromUOM, &x.ToUOM, &x.Factor,
				&v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.ProductID, x.Validity = ds(product), v.toDomain()
			return x, err
		},
	}}
}

func (m masterData) PackagingTypes() store.Repo[domain.PackagingType] {
	return pgRepo[domain.PackagingType]{s: m.s, sp: pgSpec[domain.PackagingType]{
		name: "packaging type", table: "packaging_types", codeCol: "code",
		cols:     []string{"code", "name", "net_weight_kg", "material_id", "valid_from", "valid_to", "active"},
		textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(x *domain.PackagingType) *string { return &x.ID },
		code:  func(x domain.PackagingType) string { return x.Code },
		audit: func(x *domain.PackagingType) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.PackagingType) []any {
			return []any{x.Code, x.Name, x.NetWeightKg, nu(x.MaterialID), nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.PackagingType, error) {
			var x domain.PackagingType
			var v validityCols
			var material *string
			err := r.Scan(&x.ID, &x.Code, &x.Name, &x.NetWeightKg, &material,
				&v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.MaterialID, x.Validity = ds(material), v.toDomain()
			return x, err
		},
	}}
}

func (m masterData) Warehouses() store.Repo[domain.Warehouse] {
	return pgRepo[domain.Warehouse]{s: m.s, sp: pgSpec[domain.Warehouse]{
		name: "warehouse", table: "warehouses", codeCol: "code",
		cols: []string{"factory_id", "code", "name", "storage_class", "capacity_tons", "usable_pct",
			"opening_balance", "is_silo", "valid_from", "valid_to", "active"},
		parentCol: "factory_id", textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(x *domain.Warehouse) *string { return &x.ID },
		code:  func(x domain.Warehouse) string { return x.Code },
		audit: func(x *domain.Warehouse) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.Warehouse) []any {
			return []any{x.FactoryID, x.Code, x.Name, string(x.StorageClass), x.CapacityTons,
				x.UsablePct, x.OpeningBalance, x.IsSilo, nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.Warehouse, error) {
			var x domain.Warehouse
			var v validityCols
			var class string
			err := r.Scan(&x.ID, &x.FactoryID, &x.Code, &x.Name, &class, &x.CapacityTons,
				&x.UsablePct, &x.OpeningBalance, &x.IsSilo, &v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.StorageClass, x.Validity = domain.StorageClass(class), v.toDomain()
			return x, err
		},
	}}
}

func (m masterData) Customers() store.Repo[domain.Customer] {
	return pgRepo[domain.Customer]{s: m.s, sp: pgSpec[domain.Customer]{
		name: "customer", table: "customers", codeCol: "code",
		cols:     []string{"code", "name", "country", "valid_from", "valid_to", "active"},
		textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(x *domain.Customer) *string { return &x.ID },
		code:  func(x domain.Customer) string { return x.Code },
		audit: func(x *domain.Customer) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.Customer) []any {
			return []any{x.Code, x.Name, x.Country, nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.Customer, error) {
			var x domain.Customer
			var v validityCols
			var country *string
			err := r.Scan(&x.ID, &x.Code, &x.Name, &country, &v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.Country, x.Validity = ds(country), v.toDomain()
			return x, err
		},
	}}
}

func (m masterData) Channels() store.Repo[domain.ShipmentChannel] {
	return pgRepo[domain.ShipmentChannel]{s: m.s, sp: pgSpec[domain.ShipmentChannel]{
		name: "shipment channel", table: "shipment_channels", codeCol: "code",
		cols:      []string{"code", "name", "customer_id", "category", "valid_from", "valid_to", "active"},
		parentCol: "customer_id", textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(x *domain.ShipmentChannel) *string { return &x.ID },
		code:  func(x domain.ShipmentChannel) string { return x.Code },
		audit: func(x *domain.ShipmentChannel) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.ShipmentChannel) []any {
			return []any{x.Code, x.Name, nu(x.CustomerID), x.Category, nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.ShipmentChannel, error) {
			var x domain.ShipmentChannel
			var v validityCols
			var customer *string
			err := r.Scan(&x.ID, &x.Code, &x.Name, &customer, &x.Category,
				&v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.CustomerID, x.Validity = ds(customer), v.toDomain()
			return x, err
		},
	}}
}

func (m masterData) Materials() store.Repo[domain.Material] {
	return pgRepo[domain.Material]{s: m.s, sp: pgSpec[domain.Material]{
		name: "material", table: "materials", codeCol: "code",
		cols: []string{"code", "name", "uom", "safety_stock", "lead_time_days", "scrap_pct",
			"on_hand", "on_order", "valid_from", "valid_to", "active"},
		textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(x *domain.Material) *string { return &x.ID },
		code:  func(x domain.Material) string { return x.Code },
		audit: func(x *domain.Material) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.Material) []any {
			return []any{x.Code, x.Name, x.UOM, x.SafetyStock, x.LeadTimeDays, x.ScrapPct,
				x.OnHand, x.OnOrder, nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.Material, error) {
			var x domain.Material
			var v validityCols
			err := r.Scan(&x.ID, &x.Code, &x.Name, &x.UOM, &x.SafetyStock, &x.LeadTimeDays,
				&x.ScrapPct, &x.OnHand, &x.OnOrder, &v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.Validity = v.toDomain()
			return x, err
		},
	}}
}

func (m masterData) ReasonCodes() store.Repo[domain.ReasonCode] {
	return pgRepo[domain.ReasonCode]{s: m.s, sp: pgSpec[domain.ReasonCode]{
		name: "reason code", table: "reason_codes", codeCol: "code",
		cols:      []string{"code", "name", "category", "valid_from", "valid_to", "active"},
		parentCol: "category", textCols: []string{"code", "name"}, activeCol: "active",
		id:    func(x *domain.ReasonCode) *string { return &x.ID },
		code:  func(x domain.ReasonCode) string { return x.Code },
		audit: func(x *domain.ReasonCode) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.ReasonCode) []any {
			return []any{x.Code, x.Name, x.Category, nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.ReasonCode, error) {
			var x domain.ReasonCode
			var v validityCols
			err := r.Scan(&x.ID, &x.Code, &x.Name, &x.Category, &v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.Validity = v.toDomain()
			return x, err
		},
	}}
}

// ---------------------------------------------------------------------------
// Packaging bill of materials
// ---------------------------------------------------------------------------

const bomCols = `id, packaging_id, material_id, qty_per_package, active,
	created_at, created_by, updated_at, updated_by, row_version`

func scanBOM(r scanner) (domain.PackagingBOMLine, error) {
	var line domain.PackagingBOMLine
	err := r.Scan(&line.ID, &line.PackagingID, &line.MaterialID, &line.QtyPerPackage,
		&line.Active, &line.CreatedAt, &line.CreatedBy, &line.UpdatedAt, &line.UpdatedBy,
		&line.RowVersion)
	return line, err
}

func (m masterData) ListPackagingBOM(ctx context.Context, packagingID string) ([]domain.PackagingBOMLine, error) {
	query := `SELECT ` + bomCols + ` FROM packaging_bom`
	args := []any{}
	if packagingID != "" {
		query += ` WHERE packaging_id = $1`
		args = append(args, packagingID)
	}
	query += ` ORDER BY packaging_id, material_id`

	rows, err := m.s.q.Query(ctx, query, args...)
	if err != nil {
		return nil, mapError("packaging bill of materials", err)
	}
	defer rows.Close()

	out := []domain.PackagingBOMLine{}
	for rows.Next() {
		line, err := scanBOM(rows)
		if err != nil {
			return nil, mapError("packaging bill of materials", err)
		}
		out = append(out, line)
	}
	return out, mapError("packaging bill of materials", rows.Err())
}

func (m masterData) SavePackagingBOM(ctx context.Context, line domain.PackagingBOMLine,
	actor string,
) (domain.PackagingBOMLine, error) {

	now := nowUTC()
	if line.ID == "" {
		line.ID = uuid.NewString()
		saved, err := scanBOM(m.s.q.QueryRow(ctx, `INSERT INTO packaging_bom
			(id, packaging_id, material_id, qty_per_package, active,
			 created_at, created_by, updated_at, updated_by, row_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$6,$7,1)
			RETURNING `+bomCols,
			line.ID, line.PackagingID, line.MaterialID, line.QtyPerPackage, line.Active,
			now, actor))
		return saved, mapError("packaging bill of materials", err)
	}

	saved, err := scanBOM(m.s.q.QueryRow(ctx, `UPDATE packaging_bom
		SET packaging_id = $1, material_id = $2, qty_per_package = $3, active = $4,
		    updated_at = $5, updated_by = $6, row_version = row_version + 1
		WHERE id = $7 AND ($8 = 0 OR row_version = $8)
		RETURNING `+bomCols,
		line.PackagingID, line.MaterialID, line.QtyPerPackage, line.Active,
		now, actor, line.ID, line.RowVersion))
	if errors.Is(err, pgx.ErrNoRows) {
		// Either the line is gone or somebody else changed it. Telling the two
		// apart costs a second query and changes nothing the caller does.
		return domain.PackagingBOMLine{}, fmt.Errorf(
			"%w: packaging bill of materials line %s is missing or was changed by somebody else",
			domain.ErrConflict, line.ID)
	}
	return saved, mapError("packaging bill of materials", err)
}

func (m masterData) DeletePackagingBOM(ctx context.Context, id string) error {
	tag, err := m.s.q.Exec(ctx, `DELETE FROM packaging_bom WHERE id = $1`, id)
	if err != nil {
		return mapError("packaging bill of materials", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: packaging bill of materials line %s", domain.ErrNotFound, id)
	}
	return nil
}

func (m masterData) CaneSources() store.Repo[domain.CaneSource] {
	return pgRepo[domain.CaneSource]{s: m.s, sp: pgSpec[domain.CaneSource]{
		name: "cane source", table: "cane_sources", codeCol: "code",
		cols: []string{"factory_id", "code", "name", "source_type", "zone", "distance_km",
			"hectares", "variety", "expected_yield_tph", "expected_pol_pct",
			"truck_capacity_tons", "trucks_per_day", "valid_from", "valid_to", "active"},
		parentCol: "factory_id", textCols: []string{"code", "name", "zone"}, activeCol: "active",
		id:    func(x *domain.CaneSource) *string { return &x.ID },
		code:  func(x domain.CaneSource) string { return x.Code },
		audit: func(x *domain.CaneSource) *domain.AuditFields { return &x.AuditFields },
		values: func(x domain.CaneSource) []any {
			return []any{x.FactoryID, x.Code, x.Name, string(x.Type), x.Zone, x.DistanceKm,
				x.Hectares, x.Variety, x.ExpectedYieldTPH, x.ExpectedPolPct,
				x.TruckCapacityTons, x.TrucksPerDay, nd(x.ValidFrom), nd(x.ValidTo), x.Active}
		},
		scan: func(r scanner) (domain.CaneSource, error) {
			var x domain.CaneSource
			var v validityCols
			var kind string
			err := r.Scan(&x.ID, &x.FactoryID, &x.Code, &x.Name, &kind, &x.Zone, &x.DistanceKm,
				&x.Hectares, &x.Variety, &x.ExpectedYieldTPH, &x.ExpectedPolPct,
				&x.TruckCapacityTons, &x.TrucksPerDay, &v.from, &v.to, &v.active,
				&x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.UpdatedBy, &x.RowVersion)
			x.Type, x.Validity = domain.SourceType(kind), v.toDomain()
			return x, err
		},
	}}
}
