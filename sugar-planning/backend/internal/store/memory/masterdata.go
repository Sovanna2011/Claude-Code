package memory

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// masterData binds each entity type to the generic repository. The closures
// below are the only entity-specific code the repository needs.
type masterData struct{ s *Store }

// MasterData returns the master-data repositories.
func (s *Store) MasterData() store.MasterData { return masterData{s} }

func (m masterData) Companies() store.Repo[domain.Company] {
	return memRepo[domain.Company]{s: m.s,
		sel: func(d *data) map[string]domain.Company { return d.companies },
		sp: spec[domain.Company]{
			name:   "company",
			id:     func(c *domain.Company) *string { return &c.ID },
			code:   func(c domain.Company) string { return c.Code },
			audit:  func(c *domain.Company) *domain.AuditFields { return &c.AuditFields },
			active: func(c *domain.Company) *bool { return &c.Active },
			text:   func(c domain.Company) string { return c.Code + " " + c.Name },
		}}
}

func (m masterData) Factories() store.Repo[domain.Factory] {
	return memRepo[domain.Factory]{s: m.s,
		sel: func(d *data) map[string]domain.Factory { return d.factories },
		sp: spec[domain.Factory]{
			name:   "factory",
			id:     func(f *domain.Factory) *string { return &f.ID },
			code:   func(f domain.Factory) string { return f.Code },
			audit:  func(f *domain.Factory) *domain.AuditFields { return &f.AuditFields },
			active: func(f *domain.Factory) *bool { return &f.Active },
			parent: func(f domain.Factory) string { return f.CompanyID },
			text:   func(f domain.Factory) string { return f.Code + " " + f.Name },
		}}
}

func (m masterData) Lines() store.Repo[domain.ProductionLine] {
	return memRepo[domain.ProductionLine]{s: m.s,
		sel: func(d *data) map[string]domain.ProductionLine { return d.lines },
		sp: spec[domain.ProductionLine]{
			name:   "production line",
			id:     func(l *domain.ProductionLine) *string { return &l.ID },
			code:   func(l domain.ProductionLine) string { return l.Code },
			audit:  func(l *domain.ProductionLine) *domain.AuditFields { return &l.AuditFields },
			active: func(l *domain.ProductionLine) *bool { return &l.Active },
			parent: func(l domain.ProductionLine) string { return l.FactoryID },
			text:   func(l domain.ProductionLine) string { return l.Code + " " + l.Name },
		}}
}

func (m masterData) Shifts() store.Repo[domain.Shift] {
	return memRepo[domain.Shift]{s: m.s,
		sel: func(d *data) map[string]domain.Shift { return d.shifts },
		sp: spec[domain.Shift]{
			name:   "shift",
			id:     func(x *domain.Shift) *string { return &x.ID },
			code:   func(x domain.Shift) string { return x.Code },
			audit:  func(x *domain.Shift) *domain.AuditFields { return &x.AuditFields },
			active: func(x *domain.Shift) *bool { return &x.Active },
			parent: func(x domain.Shift) string { return x.FactoryID },
			text:   func(x domain.Shift) string { return x.Code + " " + x.Name },
		}}
}

func (m masterData) ProductCategories() store.Repo[domain.ProductCategory] {
	return memRepo[domain.ProductCategory]{s: m.s,
		sel: func(d *data) map[string]domain.ProductCategory { return d.categories },
		sp: spec[domain.ProductCategory]{
			name:   "product category",
			id:     func(x *domain.ProductCategory) *string { return &x.ID },
			code:   func(x domain.ProductCategory) string { return x.Code },
			audit:  func(x *domain.ProductCategory) *domain.AuditFields { return &x.AuditFields },
			active: func(x *domain.ProductCategory) *bool { return &x.Active },
			text:   func(x domain.ProductCategory) string { return x.Code + " " + x.Name },
		}}
}

func (m masterData) Products() store.Repo[domain.Product] {
	return memRepo[domain.Product]{s: m.s,
		sel: func(d *data) map[string]domain.Product { return d.products },
		sp: spec[domain.Product]{
			name:   "product",
			id:     func(x *domain.Product) *string { return &x.ID },
			code:   func(x domain.Product) string { return x.Code },
			audit:  func(x *domain.Product) *domain.AuditFields { return &x.AuditFields },
			active: func(x *domain.Product) *bool { return &x.Active },
			parent: func(x domain.Product) string { return x.CategoryCode },
			text:   func(x domain.Product) string { return x.Code + " " + x.Name },
		}}
}

func (m masterData) UOMs() store.Repo[domain.UnitOfMeasure] {
	return memRepo[domain.UnitOfMeasure]{s: m.s,
		sel: func(d *data) map[string]domain.UnitOfMeasure { return d.uoms },
		sp: spec[domain.UnitOfMeasure]{
			name:   "unit of measure",
			id:     func(x *domain.UnitOfMeasure) *string { return &x.ID },
			code:   func(x domain.UnitOfMeasure) string { return x.Code },
			audit:  func(x *domain.UnitOfMeasure) *domain.AuditFields { return &x.AuditFields },
			active: func(x *domain.UnitOfMeasure) *bool { return &x.Active },
			text:   func(x domain.UnitOfMeasure) string { return x.Code + " " + x.Name },
		}}
}

func (m masterData) UOMConversions() store.Repo[domain.UOMConversion] {
	return memRepo[domain.UOMConversion]{s: m.s,
		sel: func(d *data) map[string]domain.UOMConversion { return d.uomConv },
		sp: spec[domain.UOMConversion]{
			name:   "unit conversion",
			id:     func(x *domain.UOMConversion) *string { return &x.ID },
			code:   func(x domain.UOMConversion) string { return x.ProductID + "/" + x.FromUOM + "/" + x.ToUOM },
			audit:  func(x *domain.UOMConversion) *domain.AuditFields { return &x.AuditFields },
			active: func(x *domain.UOMConversion) *bool { return &x.Active },
			parent: func(x domain.UOMConversion) string { return x.ProductID },
			text:   func(x domain.UOMConversion) string { return x.FromUOM + " " + x.ToUOM },
		}}
}

func (m masterData) PackagingTypes() store.Repo[domain.PackagingType] {
	return memRepo[domain.PackagingType]{s: m.s,
		sel: func(d *data) map[string]domain.PackagingType { return d.packaging },
		sp: spec[domain.PackagingType]{
			name:   "packaging type",
			id:     func(x *domain.PackagingType) *string { return &x.ID },
			code:   func(x domain.PackagingType) string { return x.Code },
			audit:  func(x *domain.PackagingType) *domain.AuditFields { return &x.AuditFields },
			active: func(x *domain.PackagingType) *bool { return &x.Active },
			text:   func(x domain.PackagingType) string { return x.Code + " " + x.Name },
		}}
}

func (m masterData) Warehouses() store.Repo[domain.Warehouse] {
	return memRepo[domain.Warehouse]{s: m.s,
		sel: func(d *data) map[string]domain.Warehouse { return d.warehouses },
		sp: spec[domain.Warehouse]{
			name:   "warehouse",
			id:     func(x *domain.Warehouse) *string { return &x.ID },
			code:   func(x domain.Warehouse) string { return x.Code },
			audit:  func(x *domain.Warehouse) *domain.AuditFields { return &x.AuditFields },
			active: func(x *domain.Warehouse) *bool { return &x.Active },
			parent: func(x domain.Warehouse) string { return x.FactoryID },
			text:   func(x domain.Warehouse) string { return x.Code + " " + x.Name },
		}}
}

func (m masterData) Customers() store.Repo[domain.Customer] {
	return memRepo[domain.Customer]{s: m.s,
		sel: func(d *data) map[string]domain.Customer { return d.customers },
		sp: spec[domain.Customer]{
			name:   "customer",
			id:     func(x *domain.Customer) *string { return &x.ID },
			code:   func(x domain.Customer) string { return x.Code },
			audit:  func(x *domain.Customer) *domain.AuditFields { return &x.AuditFields },
			active: func(x *domain.Customer) *bool { return &x.Active },
			text:   func(x domain.Customer) string { return x.Code + " " + x.Name },
		}}
}

func (m masterData) Channels() store.Repo[domain.ShipmentChannel] {
	return memRepo[domain.ShipmentChannel]{s: m.s,
		sel: func(d *data) map[string]domain.ShipmentChannel { return d.channels },
		sp: spec[domain.ShipmentChannel]{
			name:   "shipment channel",
			id:     func(x *domain.ShipmentChannel) *string { return &x.ID },
			code:   func(x domain.ShipmentChannel) string { return x.Code },
			audit:  func(x *domain.ShipmentChannel) *domain.AuditFields { return &x.AuditFields },
			active: func(x *domain.ShipmentChannel) *bool { return &x.Active },
			parent: func(x domain.ShipmentChannel) string { return x.CustomerID },
			text:   func(x domain.ShipmentChannel) string { return x.Code + " " + x.Name },
		}}
}

func (m masterData) Materials() store.Repo[domain.Material] {
	return memRepo[domain.Material]{s: m.s,
		sel: func(d *data) map[string]domain.Material { return d.materials },
		sp: spec[domain.Material]{
			name:   "material",
			id:     func(x *domain.Material) *string { return &x.ID },
			code:   func(x domain.Material) string { return x.Code },
			audit:  func(x *domain.Material) *domain.AuditFields { return &x.AuditFields },
			active: func(x *domain.Material) *bool { return &x.Active },
			text:   func(x domain.Material) string { return x.Code + " " + x.Name },
		}}
}

func (m masterData) ReasonCodes() store.Repo[domain.ReasonCode] {
	return memRepo[domain.ReasonCode]{s: m.s,
		sel: func(d *data) map[string]domain.ReasonCode { return d.reasons },
		sp: spec[domain.ReasonCode]{
			name:   "reason code",
			id:     func(x *domain.ReasonCode) *string { return &x.ID },
			code:   func(x domain.ReasonCode) string { return x.Code },
			audit:  func(x *domain.ReasonCode) *domain.AuditFields { return &x.AuditFields },
			active: func(x *domain.ReasonCode) *bool { return &x.Active },
			parent: func(x domain.ReasonCode) string { return x.Category },
			text:   func(x domain.ReasonCode) string { return x.Code + " " + x.Name },
		}}
}

// ---------------------------------------------------------------------------
// Packaging bill of materials
// ---------------------------------------------------------------------------

func (m masterData) ListPackagingBOM(_ context.Context, packagingID string) ([]domain.PackagingBOMLine, error) {
	m.s.lock()
	defer m.s.unlock()

	out := []domain.PackagingBOMLine{}
	for _, line := range m.s.d.packagingBOM {
		if packagingID != "" && line.PackagingID != packagingID {
			continue
		}
		out = append(out, line)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PackagingID != out[j].PackagingID {
			return out[i].PackagingID < out[j].PackagingID
		}
		return out[i].MaterialID < out[j].MaterialID
	})
	return out, nil
}

func (m masterData) SavePackagingBOM(_ context.Context, line domain.PackagingBOMLine,
	actor string,
) (domain.PackagingBOMLine, error) {

	m.s.lock()
	defer m.s.unlock()

	now := nowUTC()
	for _, existing := range m.s.d.packagingBOM {
		if existing.ID == line.ID || existing.PackagingID != line.PackagingID ||
			existing.MaterialID != line.MaterialID {
			continue
		}
		return domain.PackagingBOMLine{}, fmt.Errorf(
			"%w: that material is already on this packaging type's bill of materials",
			domain.ErrDuplicate)
	}

	if line.ID == "" {
		line.ID = uuid.NewString()
		line.CreatedAt, line.CreatedBy, line.RowVersion = now, actor, 1
	} else {
		stored, ok := m.s.d.packagingBOM[line.ID]
		if !ok {
			return domain.PackagingBOMLine{}, fmt.Errorf(
				"%w: packaging bill of materials line %s", domain.ErrNotFound, line.ID)
		}
		if line.RowVersion != 0 && line.RowVersion != stored.RowVersion {
			return domain.PackagingBOMLine{}, fmt.Errorf(
				"%w: the line is at version %d, you have %d",
				domain.ErrConflict, stored.RowVersion, line.RowVersion)
		}
		line.CreatedAt, line.CreatedBy = stored.CreatedAt, stored.CreatedBy
		line.RowVersion = stored.RowVersion + 1
	}
	line.UpdatedAt, line.UpdatedBy = now, actor
	m.s.d.packagingBOM[line.ID] = line
	return line, nil
}

func (m masterData) DeletePackagingBOM(_ context.Context, id string) error {
	m.s.lock()
	defer m.s.unlock()
	if _, ok := m.s.d.packagingBOM[id]; !ok {
		return fmt.Errorf("%w: packaging bill of materials line %s", domain.ErrNotFound, id)
	}
	delete(m.s.d.packagingBOM, id)
	return nil
}
