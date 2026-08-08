package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sovanna2011/farm-area/backend/internal/database"
	"github.com/sovanna2011/farm-area/backend/internal/domain"
)

// DashboardRepository answers every read the dashboard makes. All of them start from the same
// block_area() call, so the KPI cards, the charts, the tree and the map cannot disagree.
type DashboardRepository struct{ db *database.DB }

func NewDashboardRepository(db *database.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// areaColumns is the projection every area query shares.
const areaColumns = `
    COALESCE(SUM(ba.total_area_ha), 0),
    COALESCE(SUM(ba.plantable_area_ha), 0),
    COALESCE(SUM(ba.non_plantable_area_ha), 0),
    COALESCE(SUM(ba.new_planting_area_ha), 0),
    COALESCE(SUM(ba.ratoon_area_ha), 0),
    COALESCE(SUM(ba.area_with_cane_ha), 0),
    COALESCE(SUM(ba.available_area_ha), 0)`

// Totals is the estate-level roll-up behind the KPI cards.
func (r *DashboardRepository) Totals(ctx context.Context, f domain.Filter) (domain.Areas, int, error) {
	b := areaSource(f, "ba")
	b.applyScope(f, "ba")

	query := fmt.Sprintf(`SELECT %s, COUNT(*) FROM %s%s`, areaColumns, b.source, b.whereSQL())

	var a domain.Areas
	var blocks int
	err := r.db.Pool().QueryRow(ctx, query, b.args...).Scan(
		&a.TotalHa, &a.PlantableHa, &a.NonPlantableHa,
		&a.NewPlantingHa, &a.RatoonHa, &a.WithCaneHa, &a.AvailableHa, &blocks)
	if err != nil {
		return domain.Areas{}, 0, fmt.Errorf("dashboard totals: %w", err)
	}
	a.Round()
	return a, blocks, nil
}

// PlannedTotals is the same roll-up over the planned figures, for the plan-versus-actual header.
func (r *DashboardRepository) PlannedTotals(ctx context.Context, f domain.Filter) (domain.Areas, error) {
	b := areaSource(f, "ba")
	b.applyScope(f, "ba")

	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(ba.total_area_ha), 0),
		       COALESCE(SUM(ba.plantable_area_ha), 0),
		       COALESCE(SUM(ba.non_plantable_area_ha), 0),
		       COALESCE(SUM(ba.planned_new_planting_area_ha), 0),
		       COALESCE(SUM(ba.planned_ratoon_area_ha), 0),
		       COALESCE(SUM(ba.planned_area_with_cane_ha), 0),
		       COALESCE(SUM(ba.planned_available_area_ha), 0)
		FROM %s%s`, b.source, b.whereSQL())

	var a domain.Areas
	err := r.db.Pool().QueryRow(ctx, query, b.args...).Scan(
		&a.TotalHa, &a.PlantableHa, &a.NonPlantableHa,
		&a.NewPlantingHa, &a.RatoonHa, &a.WithCaneHa, &a.AvailableHa)
	if err != nil {
		return domain.Areas{}, fmt.Errorf("dashboard planned totals: %w", err)
	}
	a.Round()
	return a, nil
}

// treeRow is one block, carrying its zone and farm with it. Farms and zones that have nothing
// beneath them still come back — with NULL block columns — because a zone with no blocks is part
// of the estate's structure and hiding it would make the hierarchy look smaller than it is.
type treeRow struct {
	FarmID   int
	FarmCode string
	FarmName string
	FarmLat  *float64
	FarmLng  *float64

	ZoneID   *int
	ZoneCode *string
	ZoneName *string
	ZoneLat  *float64
	ZoneLng  *float64

	BlockID    *int
	BlockCode  *string
	BlockName  *string
	BlockLat   *float64
	BlockLng   *float64
	LandStatus *string
	CaneStatus *string
	Variety    *string

	Areas domain.Areas
}

// Tree returns the farm → zone → block hierarchy with every figure summed from the blocks.
func (r *DashboardRepository) Tree(ctx context.Context, f domain.Filter) ([]*domain.TreeNode, error) {
	// The block filters have to sit in the LEFT JOIN's ON clause, not in WHERE: moved to WHERE
	// they would turn the outer join back into an inner one and drop every empty zone.
	b := areaSource(f, "ba")
	joinConds := []string{"ba.zone_id = z.id"}
	scope := &builder{args: b.args}
	scope.applyScope(f, "ba")
	joinConds = append(joinConds, scope.where...)

	scopeArgs := &builder{args: scope.args}
	subScope := plantingScope(scopeArgs, f)

	farmWhere := &builder{args: scopeArgs.args}
	if f.CompanyID != nil {
		farmWhere.eq("c.id", *f.CompanyID)
	}
	if f.PlantationID != nil {
		farmWhere.eq("p.id", *f.PlantationID)
	}
	if f.FarmID != nil {
		farmWhere.eq("f.id", *f.FarmID)
	}
	if f.ZoneID != nil {
		farmWhere.raw(fmt.Sprintf("(z.id IS NULL OR z.id = %s)", farmWhere.add(*f.ZoneID)))
	}
	if !f.IncludeInactive {
		farmWhere.raw("f.active")
	}

	query := fmt.Sprintf(`
		SELECT f.id, f.code, f.name,
		       ST_Y(ST_Centroid(f.boundary::geometry)), ST_X(ST_Centroid(f.boundary::geometry)),
		       z.id, z.code, z.name,
		       ST_Y(ST_Centroid(z.boundary::geometry)), ST_X(ST_Centroid(z.boundary::geometry)),
		       ba.block_id, ba.block_code, ba.block_name, ba.latitude, ba.longitude,
		       ba.land_status::text, ba.cane_status::text,
		       (SELECT string_agg(DISTINCT cv.name, ', ' ORDER BY cv.name)
		          FROM block_planting bp JOIN cane_variety cv ON cv.id = bp.cane_variety_id
		         WHERE bp.block_id = ba.block_id AND %s),
		       COALESCE(ba.total_area_ha, 0), COALESCE(ba.plantable_area_ha, 0),
		       COALESCE(ba.non_plantable_area_ha, 0), COALESCE(ba.new_planting_area_ha, 0),
		       COALESCE(ba.ratoon_area_ha, 0), COALESCE(ba.area_with_cane_ha, 0),
		       COALESCE(ba.available_area_ha, 0)
		  FROM farm f
		  JOIN plantation p ON p.id = f.plantation_id
		  JOIN company c    ON c.id = p.company_id
		  LEFT JOIN zone z  ON z.farm_id = f.id %s
		  LEFT JOIN %s ON %s
		 %s
		 ORDER BY f.code, z.code NULLS LAST, ba.block_code NULLS LAST`,
		subScope, zoneActiveClause(f), b.source, joinAnd(joinConds), farmWhere.whereSQL())

	rows, err := r.db.Pool().Query(ctx, query, farmWhere.args...)
	if err != nil {
		return nil, fmt.Errorf("tree query: %w", err)
	}
	defer rows.Close()

	var flat []treeRow
	for rows.Next() {
		var t treeRow
		if err := rows.Scan(
			&t.FarmID, &t.FarmCode, &t.FarmName, &t.FarmLat, &t.FarmLng,
			&t.ZoneID, &t.ZoneCode, &t.ZoneName, &t.ZoneLat, &t.ZoneLng,
			&t.BlockID, &t.BlockCode, &t.BlockName, &t.BlockLat, &t.BlockLng,
			&t.LandStatus, &t.CaneStatus, &t.Variety,
			&t.Areas.TotalHa, &t.Areas.PlantableHa, &t.Areas.NonPlantableHa,
			&t.Areas.NewPlantingHa, &t.Areas.RatoonHa, &t.Areas.WithCaneHa, &t.Areas.AvailableHa,
		); err != nil {
			return nil, fmt.Errorf("tree scan: %w", err)
		}
		flat = append(flat, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return buildTree(flat), nil
}

// plantingScope renders the planting filters as a predicate a correlated subquery can use, so the
// variety and the dates shown beside a block's areas come from the same crop year as those areas.
func plantingScope(b *builder, f domain.Filter) string {
	conds := []string{"true"}
	if f.CropYear != nil {
		conds = append(conds, "bp.crop_year = "+b.add(*f.CropYear))
	}
	if f.PlantingYear != nil {
		conds = append(conds, "bp.planting_year = "+b.add(*f.PlantingYear))
	}
	if f.SeasonID != nil {
		conds = append(conds, "bp.crop_season_id = "+b.add(*f.SeasonID))
	}
	if f.VarietyID != nil {
		conds = append(conds, "bp.cane_variety_id = "+b.add(*f.VarietyID))
	}
	if f.PlantingType != nil {
		conds = append(conds, "bp.planting_type = "+b.add(*f.PlantingType)+"::planting_type")
	}
	return joinAnd(conds)
}

func zoneActiveClause(f domain.Filter) string {
	if f.IncludeInactive {
		return ""
	}
	return "AND z.active"
}

func joinAnd(conds []string) string {
	out := conds[0]
	for _, c := range conds[1:] {
		out += " AND " + c
	}
	return out
}

// buildTree assembles the flat result into farm → zone → block and rolls the areas up. The
// aggregation is a plain sum of the children at each level: no node holds a figure of its own.
func buildTree(rows []treeRow) []*domain.TreeNode {
	var farms []*domain.TreeNode
	farmByID := map[int]*domain.TreeNode{}
	zoneByID := map[int]*domain.TreeNode{}

	for _, row := range rows {
		farm, ok := farmByID[row.FarmID]
		if !ok {
			farm = &domain.TreeNode{
				NodeType: "Farm", ID: row.FarmID, Code: row.FarmCode, Name: row.FarmName,
				Latitude: row.FarmLat, Longitude: row.FarmLng,
				MapURL: domain.GoogleMapsURL(row.FarmLat, row.FarmLng), Children: []*domain.TreeNode{},
			}
			farmByID[row.FarmID] = farm
			farms = append(farms, farm)
		}

		if row.ZoneID == nil {
			continue
		}
		zone, ok := zoneByID[*row.ZoneID]
		if !ok {
			zone = &domain.TreeNode{
				NodeType: "Zone", ID: *row.ZoneID, Code: deref(row.ZoneCode), Name: deref(row.ZoneName),
				Latitude: row.ZoneLat, Longitude: row.ZoneLng,
				MapURL: domain.GoogleMapsURL(row.ZoneLat, row.ZoneLng), Children: []*domain.TreeNode{},
			}
			zoneByID[*row.ZoneID] = zone
			farm.Children = append(farm.Children, zone)
		}

		if row.BlockID == nil {
			continue
		}
		block := &domain.TreeNode{
			NodeType: "Block", ID: *row.BlockID, Code: deref(row.BlockCode), Name: deref(row.BlockName),
			Areas: row.Areas, BlockCount: 1,
			LandStatus: deref(row.LandStatus), CaneStatus: deref(row.CaneStatus), VarietyName: deref(row.Variety),
			Latitude: row.BlockLat, Longitude: row.BlockLng,
			MapURL: domain.GoogleMapsURL(row.BlockLat, row.BlockLng), Children: []*domain.TreeNode{},
		}
		block.Areas.Round()
		setPercentages(block)
		zone.Children = append(zone.Children, block)
	}

	for _, farm := range farms {
		for _, zone := range farm.Children {
			for _, block := range zone.Children {
				zone.Areas.Add(block.Areas)
				zone.BlockCount++
			}
			zone.Areas.Round()
			setPercentages(zone)
			farm.Areas.Add(zone.Areas)
			farm.BlockCount += zone.BlockCount
		}
		farm.Areas.Round()
		setPercentages(farm)
	}
	if farms == nil {
		farms = []*domain.TreeNode{}
	}
	return farms
}

func setPercentages(n *domain.TreeNode) {
	n.PercentWithCane = domain.PercentOfTotal(n.Areas.WithCaneHa, n.Areas.TotalHa)
	n.PercentAvailable = domain.PercentOfTotal(n.Areas.AvailableHa, n.Areas.TotalHa)
	n.PercentCannotPlan = domain.PercentOfTotal(n.Areas.NonPlantableHa, n.Areas.TotalHa)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// AreaByLevel powers the "area by farm" and "area by zone" charts.
func (r *DashboardRepository) AreaByLevel(ctx context.Context, f domain.Filter, level string) ([]domain.ChartPoint, error) {
	idCol, nameCol := "ba.farm_id", "ba.farm_name"
	if level == "zone" {
		idCol, nameCol = "ba.zone_id", "ba.zone_name"
	}

	b := areaSource(f, "ba")
	b.applyScope(f, "ba")
	query := fmt.Sprintf(`
		SELECT %s, %s, COALESCE(SUM(ba.total_area_ha), 0), COALESCE(SUM(ba.area_with_cane_ha), 0)
		  FROM %s%s
		 GROUP BY %s, %s
		 ORDER BY %s`, idCol, nameCol, b.source, b.whereSQL(), idCol, nameCol, nameCol)

	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return nil, fmt.Errorf("area by %s: %w", level, err)
	}
	defer rows.Close()

	points := []domain.ChartPoint{}
	for rows.Next() {
		var p domain.ChartPoint
		var withCane float64
		if err := rows.Scan(&p.ID, &p.Category, &p.Value, &withCane); err != nil {
			return nil, err
		}
		p.Extra = &withCane
		points = append(points, p)
	}
	return points, rows.Err()
}

// MonthlyProgress is the planting progress chart: planned and actual hectares by the month the
// work was due and the month it happened.
func (r *DashboardRepository) MonthlyProgress(ctx context.Context, f domain.Filter) ([]domain.ChartPoint, error) {
	b := areaSource(f, "ba")
	b.applyScope(f, "ba")

	// The planting rows are re-filtered here because a month has to come from a date on the
	// planting record, which block_area() has already aggregated away.
	pf := &builder{args: b.args}
	pf.raw("bp.block_id = ba.block_id")
	if f.CropYear != nil {
		pf.eq("bp.crop_year", *f.CropYear)
	}
	if f.PlantingYear != nil {
		pf.eq("bp.planting_year", *f.PlantingYear)
	}
	if f.SeasonID != nil {
		pf.eq("bp.crop_season_id", *f.SeasonID)
	}
	if f.VarietyID != nil {
		pf.eq("bp.cane_variety_id", *f.VarietyID)
	}
	if f.PlantingType != nil {
		pf.raw(fmt.Sprintf("bp.planting_type = %s::planting_type", pf.add(*f.PlantingType)))
	}

	query := fmt.Sprintf(`
		WITH scoped AS (
		    SELECT bp.planned_date, bp.actual_date, bp.planned_area_ha, bp.actual_area_ha
		      FROM %s
		      JOIN block_planting bp ON %s
		     %s
		)
		SELECT m.month,
		       COALESCE((SELECT SUM(planned_area_ha) FROM scoped
		                  WHERE planned_date IS NOT NULL AND EXTRACT(MONTH FROM planned_date) = m.month), 0),
		       COALESCE((SELECT SUM(actual_area_ha) FROM scoped
		                  WHERE actual_date IS NOT NULL AND EXTRACT(MONTH FROM actual_date) = m.month), 0)
		  FROM generate_series(1, 12) AS m(month)
		 ORDER BY m.month`,
		b.source, joinAnd(pf.where), b.whereSQL())

	rows, err := r.db.Pool().Query(ctx, query, pf.args...)
	if err != nil {
		return nil, fmt.Errorf("monthly progress: %w", err)
	}
	defer rows.Close()

	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	points := []domain.ChartPoint{}
	for rows.Next() {
		var month int
		var planned, actual float64
		if err := rows.Scan(&month, &planned, &actual); err != nil {
			return nil, err
		}
		p := domain.ChartPoint{Category: months[month-1], Value: actual, Extra: &planned}
		points = append(points, p)
	}
	return points, rows.Err()
}

// PlanVsActual groups the programme against what happened, at the level the caller asks for.
func (r *DashboardRepository) PlanVsActual(ctx context.Context, f domain.Filter, level string) ([]domain.PlanActualRow, error) {
	var idCol, codeCol, nameCol string
	switch level {
	case "zone":
		idCol, codeCol, nameCol = "ba.zone_id", "ba.zone_code", "ba.zone_name"
	case "block":
		idCol, codeCol, nameCol = "ba.block_id", "ba.block_code", "ba.block_name"
	default:
		level = "farm"
		idCol, codeCol, nameCol = "ba.farm_id", "ba.farm_code", "ba.farm_name"
	}

	b := areaSource(f, "ba")
	b.applyScope(f, "ba")
	query := fmt.Sprintf(`
		SELECT %s, %s, %s,
		       COALESCE(SUM(ba.planned_new_planting_area_ha), 0), COALESCE(SUM(ba.new_planting_area_ha), 0),
		       COALESCE(SUM(ba.planned_ratoon_area_ha), 0),       COALESCE(SUM(ba.ratoon_area_ha), 0),
		       COALESCE(SUM(ba.planned_area_with_cane_ha), 0),    COALESCE(SUM(ba.area_with_cane_ha), 0),
		       COALESCE(SUM(ba.planned_available_area_ha), 0),    COALESCE(SUM(ba.available_area_ha), 0)
		  FROM %s%s
		 GROUP BY %s, %s, %s
		 ORDER BY %s`, idCol, codeCol, nameCol, b.source, b.whereSQL(), idCol, codeCol, nameCol, codeCol)

	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return nil, fmt.Errorf("plan vs actual: %w", err)
	}
	defer rows.Close()

	out := []domain.PlanActualRow{}
	for rows.Next() {
		row := domain.PlanActualRow{Level: level}
		if err := rows.Scan(&row.ID, &row.Code, &row.Name,
			&row.PlannedNewHa, &row.ActualNewHa, &row.PlannedRatoonHa, &row.ActualRatoonHa,
			&row.PlannedWithCaneHa, &row.ActualWithCaneHa, &row.PlannedAvailableHa, &row.ActualAvailableHa); err != nil {
			return nil, err
		}
		if f.CropYear != nil {
			row.CropYear = *f.CropYear
		}
		row.VarianceHa = round4(row.ActualWithCaneHa - row.PlannedWithCaneHa)
		row.AchievementPercent = domain.Achievement(row.ActualWithCaneHa, row.PlannedWithCaneHa)
		out = append(out, row)
	}
	return out, rows.Err()
}

// MonthlyPlanVsActual is the same comparison broken down by the month the work happened, which is
// what the specification's "Year → Month" drill-down reads.
func (r *DashboardRepository) MonthlyPlanVsActual(ctx context.Context, f domain.Filter) ([]domain.PlanActualRow, error) {
	b := areaSource(f, "ba")
	b.applyScope(f, "ba")

	pf := &builder{args: b.args}
	pf.raw("bp.block_id = ba.block_id")
	// A row with neither date belongs to no month, so it is excluded in the join rather than in a
	// HAVING clause — Postgres will not let HAVING reach a column the GROUP BY does not carry.
	pf.raw("(bp.actual_date IS NOT NULL OR bp.planned_date IS NOT NULL)")
	if f.CropYear != nil {
		pf.eq("bp.crop_year", *f.CropYear)
	}
	if f.SeasonID != nil {
		pf.eq("bp.crop_season_id", *f.SeasonID)
	}
	if f.PlantingType != nil {
		pf.raw(fmt.Sprintf("bp.planting_type = %s::planting_type", pf.add(*f.PlantingType)))
	}

	query := fmt.Sprintf(`
		SELECT COALESCE(EXTRACT(MONTH FROM bp.actual_date), EXTRACT(MONTH FROM bp.planned_date))::int AS month,
		       bp.crop_year,
		       COALESCE(SUM(bp.planned_area_ha) FILTER (WHERE bp.planting_type = 'NewPlanting'), 0),
		       COALESCE(SUM(bp.actual_area_ha)  FILTER (WHERE bp.planting_type = 'NewPlanting'), 0),
		       COALESCE(SUM(bp.planned_area_ha) FILTER (WHERE bp.planting_type = 'Ratoon'), 0),
		       COALESCE(SUM(bp.actual_area_ha)  FILTER (WHERE bp.planting_type = 'Ratoon'), 0)
		  FROM %s
		  JOIN block_planting bp ON %s
		 %s
		 GROUP BY 1, 2
		 ORDER BY 2, 1`, b.source, joinAnd(pf.where), b.whereSQL())

	rows, err := r.db.Pool().Query(ctx, query, pf.args...)
	if err != nil {
		return nil, fmt.Errorf("monthly plan vs actual: %w", err)
	}
	defer rows.Close()

	months := []string{"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}

	out := []domain.PlanActualRow{}
	for rows.Next() {
		row := domain.PlanActualRow{Level: "month"}
		if err := rows.Scan(&row.Month, &row.CropYear,
			&row.PlannedNewHa, &row.ActualNewHa, &row.PlannedRatoonHa, &row.ActualRatoonHa); err != nil {
			return nil, err
		}
		row.Code = fmt.Sprintf("%d-%02d", row.CropYear, row.Month)
		row.Name = months[row.Month-1]
		row.PlannedWithCaneHa = round4(row.PlannedNewHa + row.PlannedRatoonHa)
		row.ActualWithCaneHa = round4(row.ActualNewHa + row.ActualRatoonHa)
		row.VarianceHa = round4(row.ActualWithCaneHa - row.PlannedWithCaneHa)
		row.AchievementPercent = domain.Achievement(row.ActualWithCaneHa, row.PlannedWithCaneHa)
		out = append(out, row)
	}
	return out, rows.Err()
}

// MapData returns the blocks as GeoJSON with their figures attached, plus the farm and zone
// outlines, so one request draws the whole map and fills the detail panel on click.
func (r *DashboardRepository) MapData(ctx context.Context, f domain.Filter) (domain.MapFeatureCollection, error) {
	b := areaSource(f, "ba")
	b.applyScope(f, "ba")
	subScope := plantingScope(b, f)

	query := fmt.Sprintf(`
		SELECT ba.block_id, ba.block_code, ba.block_name,
		       ba.farm_id, ba.farm_name, ba.zone_id, ba.zone_name,
		       ba.total_area_ha, ba.plantable_area_ha, ba.non_plantable_area_ha,
		       ba.new_planting_area_ha, ba.ratoon_area_ha, ba.area_with_cane_ha, ba.available_area_ha,
		       ba.land_status::text, ba.cane_status::text, ba.latitude, ba.longitude,
		       ST_AsGeoJSON(bl.boundary::geometry),
		       (SELECT string_agg(DISTINCT cv.name, ', ' ORDER BY cv.name)
		          FROM block_planting bp JOIN cane_variety cv ON cv.id = bp.cane_variety_id
		         WHERE bp.block_id = ba.block_id AND %[2]s),
		       (SELECT min(bp.actual_date)::text FROM block_planting bp
		         WHERE bp.block_id = ba.block_id AND %[2]s),
		       (SELECT max(bp.expected_harvest_date)::text FROM block_planting bp
		         WHERE bp.block_id = ba.block_id AND %[2]s)
		  FROM %[1]s
		  JOIN block bl ON bl.id = ba.block_id
		 %[3]s
		 ORDER BY ba.block_code`, b.source, subScope, b.whereSQL())

	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return domain.MapFeatureCollection{}, fmt.Errorf("map data: %w", err)
	}
	defer rows.Close()

	features := []domain.MapFeature{}
	for rows.Next() {
		var (
			id, farmID, zoneID                          int
			code, name, farmName, zoneName              string
			total, plantable, nonPlantable              float64
			newPlanting, ratoon, withCane, available    float64
			landStatus, caneStatus                      string
			lat, lng                                    *float64
			geoJSON, variety, plantingDate, harvestDate *string
		)
		if err := rows.Scan(&id, &code, &name, &farmID, &farmName, &zoneID, &zoneName,
			&total, &plantable, &nonPlantable, &newPlanting, &ratoon, &withCane, &available,
			&landStatus, &caneStatus, &lat, &lng, &geoJSON, &variety, &plantingDate, &harvestDate); err != nil {
			return domain.MapFeatureCollection{}, err
		}

		var geometry any
		if geoJSON != nil {
			if err := json.Unmarshal([]byte(*geoJSON), &geometry); err != nil {
				return domain.MapFeatureCollection{}, fmt.Errorf("block %s geometry: %w", code, err)
			}
		} else if lat != nil && lng != nil {
			// No survey yet: a point is still enough to put the block on the map.
			geometry = map[string]any{"type": "Point", "coordinates": []float64{*lng, *lat}}
		} else {
			continue
		}

		features = append(features, domain.MapFeature{
			Type:     "Feature",
			Geometry: geometry,
			Properties: map[string]any{
				"blockId": id, "blockCode": code, "blockName": name,
				"farmId": farmID, "farmName": farmName, "zoneId": zoneID, "zoneName": zoneName,
				"totalAreaHa": total, "plantableAreaHa": plantable, "nonPlantableAreaHa": nonPlantable,
				"newPlantingAreaHa": newPlanting, "ratoonAreaHa": ratoon,
				"areaWithCaneHa": withCane, "availableAreaHa": available,
				"landStatus": landStatus, "caneStatus": caneStatus,
				"caneVarietyName": variety, "plantingDate": plantingDate, "expectedHarvestDate": harvestDate,
				"latitude": lat, "longitude": lng,
				"mapUrl": domain.GoogleMapsURL(lat, lng),
			},
		})
	}
	if err := rows.Err(); err != nil {
		return domain.MapFeatureCollection{}, err
	}
	return domain.MapFeatureCollection{Type: "FeatureCollection", Features: features}, nil
}

// Outlines returns the farm and zone boundaries, drawn under the blocks so the map shows the
// hierarchy rather than a scatter of squares.
func (r *DashboardRepository) Outlines(ctx context.Context, f domain.Filter) (domain.MapFeatureCollection, error) {
	// One argument serves both halves of the UNION: "$1 IS NULL OR id = $1" means "no filter set"
	// without the two branches needing a placeholder each.
	var farmID *int
	if f.FarmID != nil {
		farmID = f.FarmID
	}
	const query = `
		SELECT 'Farm', f.id, f.code, f.name, ST_AsGeoJSON(f.boundary::geometry)
		  FROM farm f
		 WHERE f.boundary IS NOT NULL AND ($1::int IS NULL OR f.id = $1)
		 UNION ALL
		SELECT 'Zone', z.id, z.code, z.name, ST_AsGeoJSON(z.boundary::geometry)
		  FROM zone z
		 WHERE z.boundary IS NOT NULL AND ($1::int IS NULL OR z.farm_id = $1)`

	rows, err := r.db.Pool().Query(ctx, query, farmID)
	if err != nil {
		return domain.MapFeatureCollection{}, fmt.Errorf("outlines: %w", err)
	}
	defer rows.Close()

	features := []domain.MapFeature{}
	for rows.Next() {
		var level, code, name string
		var id int
		var geoJSON *string
		if err := rows.Scan(&level, &id, &code, &name, &geoJSON); err != nil {
			return domain.MapFeatureCollection{}, err
		}
		if geoJSON == nil {
			continue
		}
		var geometry any
		if err := json.Unmarshal([]byte(*geoJSON), &geometry); err != nil {
			return domain.MapFeatureCollection{}, err
		}
		features = append(features, domain.MapFeature{
			Type:       "Feature",
			Geometry:   geometry,
			Properties: map[string]any{"level": level, "id": id, "code": code, "name": name},
		})
	}
	return domain.MapFeatureCollection{Type: "FeatureCollection", Features: features}, rows.Err()
}

func round4(v float64) float64 {
	return float64(int64(v*10000+copySign(0.5, v))) / 10000
}

func copySign(magnitude, sign float64) float64 {
	if sign < 0 {
		return -magnitude
	}
	return magnitude
}
