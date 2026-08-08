// Package repository is the only place that speaks SQL. Services depend on its interfaces, so the
// business rules can be tested without a database and the queries can be changed without touching
// them.
package repository

import (
	"fmt"
	"strings"

	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
)

// builder accumulates positional arguments so a query can be assembled from optional pieces
// without ever concatenating a caller's value into the SQL text.
type builder struct {
	args   []any
	where  []string
	source string
}

func (b *builder) add(v any) string {
	b.args = append(b.args, v)
	return fmt.Sprintf("$%d", len(b.args))
}

func (b *builder) eq(column string, value any) {
	b.where = append(b.where, fmt.Sprintf("%s = %s", column, b.add(value)))
}

func (b *builder) raw(clause string) { b.where = append(b.where, clause) }

func (b *builder) whereSQL() string {
	if len(b.where) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(b.where, " AND ")
}

// areaSource renders the call to block_area() with the planting filters bound, and returns a
// builder already carrying those five arguments. Everything that reports an area figure starts
// here, which is what stops the KPI cards and the tree report from filtering differently.
func areaSource(f domain.Filter, alias string) *builder {
	b := &builder{}
	b.source = fmt.Sprintf("block_area(%s, %s, %s, %s, %s) %s",
		b.add(f.CropYear), b.add(f.PlantingYear), b.add(f.SeasonID), b.add(f.VarietyID), b.add(f.PlantingType), alias)
	return b
}

// applyScope adds the hierarchy and status filters that apply to any query over block rows.
func (b *builder) applyScope(f domain.Filter, alias string) {
	col := func(name string) string { return alias + "." + name }

	if f.CompanyID != nil {
		b.eq(col("company_id"), *f.CompanyID)
	}
	if f.PlantationID != nil {
		b.eq(col("plantation_id"), *f.PlantationID)
	}
	if f.FarmID != nil {
		b.eq(col("farm_id"), *f.FarmID)
	}
	if f.ZoneID != nil {
		b.eq(col("zone_id"), *f.ZoneID)
	}
	if f.BlockID != nil {
		b.eq(col("block_id"), *f.BlockID)
	}
	if f.LandStatus != nil {
		b.where = append(b.where, fmt.Sprintf("%s = %s::land_status", col("land_status"), b.add(*f.LandStatus)))
	}
	if f.CaneStatus != nil {
		b.where = append(b.where, fmt.Sprintf("%s = %s::cane_status", col("cane_status"), b.add(*f.CaneStatus)))
	}
	if !f.IncludeInactive {
		b.raw(col("active"))
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		p := b.add("%" + strings.ToLower(s) + "%")
		b.raw(fmt.Sprintf(
			"(lower(%s) LIKE %s OR lower(%s) LIKE %s OR lower(%s) LIKE %s OR lower(%s) LIKE %s OR lower(%s) LIKE %s OR lower(%s) LIKE %s)",
			col("block_code"), p, col("block_name"), p,
			col("zone_code"), p, col("zone_name"), p,
			col("farm_code"), p, col("farm_name"), p))
	}
}
