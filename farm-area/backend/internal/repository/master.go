package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/sovanna2011/farm-area/backend/internal/database"
	"github.com/sovanna2011/farm-area/backend/internal/domain"
)

// MasterRepository owns the farm, zone and block master data, their geometry and their planting
// records. Every method takes a Querier so the same code runs inside a transaction and outside it.
type MasterRepository struct{ db *database.DB }

func NewMasterRepository(db *database.DB) *MasterRepository { return &MasterRepository{db: db} }

func (r *MasterRepository) q(tx database.Querier) database.Querier {
	if tx != nil {
		return tx
	}
	return r.db.Pool()
}

// ---------------------------------------------------------------- farms

const farmSelect = `
	SELECT f.id, f.plantation_id, p.name, f.code, f.name, f.manager_name, f.remark, f.active,
	       (SELECT count(*) FROM zone z WHERE z.farm_id = f.id),
	       (SELECT count(*) FROM block b JOIN zone z ON z.id = b.zone_id WHERE z.farm_id = f.id),
	       f.boundary IS NOT NULL,
	       ST_Y(ST_Centroid(f.boundary::geometry)), ST_X(ST_Centroid(f.boundary::geometry)),
	       f.version, f.updated_at, f.updated_by
	  FROM farm f JOIN plantation p ON p.id = f.plantation_id`

func scanFarm(row pgx.Row) (domain.Farm, error) {
	var f domain.Farm
	err := row.Scan(&f.ID, &f.PlantationID, &f.PlantationName, &f.Code, &f.Name, &f.ManagerName,
		&f.Remark, &f.Active, &f.ZoneCount, &f.BlockCount, &f.HasBoundary,
		&f.Latitude, &f.Longitude, &f.Version, &f.UpdatedAt, &f.UpdatedBy)
	f.MapURL = domain.GoogleMapsURL(f.Latitude, f.Longitude)
	return f, err
}

func (r *MasterRepository) ListFarms(ctx context.Context, f domain.Filter, page domain.Page) ([]domain.Farm, int, error) {
	b := &builder{}
	if f.PlantationID != nil {
		b.eq("f.plantation_id", *f.PlantationID)
	}
	if f.CompanyID != nil {
		b.eq("p.company_id", *f.CompanyID)
	}
	if f.FarmID != nil {
		b.eq("f.id", *f.FarmID)
	}
	if !f.IncludeInactive {
		b.raw("f.active")
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		p := b.add("%" + strings.ToLower(s) + "%")
		b.raw(fmt.Sprintf("(lower(f.code) LIKE %s OR lower(f.name) LIKE %s)", p, p))
	}

	var total int
	countQuery := "SELECT count(*) FROM farm f JOIN plantation p ON p.id = f.plantation_id" + b.whereSQL()
	if err := r.db.Pool().QueryRow(ctx, countQuery, b.args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count farms: %w", err)
	}

	query := farmSelect + b.whereSQL() + fmt.Sprintf(" ORDER BY f.code LIMIT %s OFFSET %s",
		b.add(page.Size), b.add(page.Offset()))
	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list farms: %w", err)
	}
	defer rows.Close()

	farms := []domain.Farm{}
	for rows.Next() {
		farm, err := scanFarm(rows)
		if err != nil {
			return nil, 0, err
		}
		farms = append(farms, farm)
	}
	return farms, total, rows.Err()
}

func (r *MasterRepository) GetFarm(ctx context.Context, tx database.Querier, id int) (domain.Farm, error) {
	farm, err := scanFarm(r.q(tx).QueryRow(ctx, farmSelect+" WHERE f.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Farm{}, domain.NotFound("Farm", id)
	}
	return farm, err
}

// FarmInput is the writable half of a farm. Note the absence of any area column: a farm's areas
// are the sum of its blocks, and there is nowhere here to override them.
type FarmInput struct {
	PlantationID int     `json:"plantationId"`
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	ManagerName  *string `json:"managerName"`
	Remark       *string `json:"remark"`
	Active       *bool   `json:"active"`
	Version      int     `json:"version"`
}

func (r *MasterRepository) CreateFarm(ctx context.Context, tx database.Querier, in FarmInput, actor string) (int, error) {
	var id int
	err := r.q(tx).QueryRow(ctx, `
		INSERT INTO farm(plantation_id, code, name, manager_name, remark, active, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6, true), $7, $7) RETURNING id`,
		in.PlantationID, in.Code, in.Name, in.ManagerName, in.Remark, in.Active, actor).Scan(&id)
	return id, mapWriteError(err, "Farm", in.Code)
}

func (r *MasterRepository) UpdateFarm(ctx context.Context, tx database.Querier, id int, in FarmInput, actor string) error {
	tag, err := r.q(tx).Exec(ctx, `
		UPDATE farm SET code = $2, name = $3, manager_name = $4, remark = $5,
		                active = COALESCE($6, active),
		                version = version + 1, updated_at = now(), updated_by = $7
		 WHERE id = $1 AND version = $8`,
		id, in.Code, in.Name, in.ManagerName, in.Remark, in.Active, actor, in.Version)
	if err != nil {
		return mapWriteError(err, "Farm", in.Code)
	}
	if tag.RowsAffected() == 0 {
		return r.explainMiss(ctx, tx, "farm", "Farm", id)
	}
	return nil
}

// ---------------------------------------------------------------- zones

const zoneSelect = `
	SELECT z.id, z.farm_id, f.name, z.code, z.name, z.supervisor_name, z.remark, z.active,
	       (SELECT count(*) FROM block b WHERE b.zone_id = z.id),
	       z.boundary IS NOT NULL,
	       ST_Y(ST_Centroid(z.boundary::geometry)), ST_X(ST_Centroid(z.boundary::geometry)),
	       z.version, z.updated_at, z.updated_by
	  FROM zone z JOIN farm f ON f.id = z.farm_id`

func scanZone(row pgx.Row) (domain.Zone, error) {
	var z domain.Zone
	err := row.Scan(&z.ID, &z.FarmID, &z.FarmName, &z.Code, &z.Name, &z.SupervisorName, &z.Remark,
		&z.Active, &z.BlockCount, &z.HasBoundary, &z.Latitude, &z.Longitude,
		&z.Version, &z.UpdatedAt, &z.UpdatedBy)
	z.MapURL = domain.GoogleMapsURL(z.Latitude, z.Longitude)
	return z, err
}

func (r *MasterRepository) ListZones(ctx context.Context, f domain.Filter, page domain.Page) ([]domain.Zone, int, error) {
	b := &builder{}
	if f.FarmID != nil {
		b.eq("z.farm_id", *f.FarmID)
	}
	if f.ZoneID != nil {
		b.eq("z.id", *f.ZoneID)
	}
	if !f.IncludeInactive {
		b.raw("z.active")
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		p := b.add("%" + strings.ToLower(s) + "%")
		b.raw(fmt.Sprintf("(lower(z.code) LIKE %s OR lower(z.name) LIKE %s)", p, p))
	}

	var total int
	if err := r.db.Pool().QueryRow(ctx,
		"SELECT count(*) FROM zone z JOIN farm f ON f.id = z.farm_id"+b.whereSQL(), b.args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count zones: %w", err)
	}

	query := zoneSelect + b.whereSQL() + fmt.Sprintf(" ORDER BY z.code LIMIT %s OFFSET %s",
		b.add(page.Size), b.add(page.Offset()))
	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list zones: %w", err)
	}
	defer rows.Close()

	zones := []domain.Zone{}
	for rows.Next() {
		z, err := scanZone(rows)
		if err != nil {
			return nil, 0, err
		}
		zones = append(zones, z)
	}
	return zones, total, rows.Err()
}

func (r *MasterRepository) GetZone(ctx context.Context, tx database.Querier, id int) (domain.Zone, error) {
	z, err := scanZone(r.q(tx).QueryRow(ctx, zoneSelect+" WHERE z.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Zone{}, domain.NotFound("Zone", id)
	}
	return z, err
}

type ZoneInput struct {
	FarmID         int     `json:"farmId"`
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	SupervisorName *string `json:"supervisorName"`
	Remark         *string `json:"remark"`
	Active         *bool   `json:"active"`
	Version        int     `json:"version"`
}

func (r *MasterRepository) CreateZone(ctx context.Context, tx database.Querier, in ZoneInput, actor string) (int, error) {
	var id int
	err := r.q(tx).QueryRow(ctx, `
		INSERT INTO zone(farm_id, code, name, supervisor_name, remark, active, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6, true), $7, $7) RETURNING id`,
		in.FarmID, in.Code, in.Name, in.SupervisorName, in.Remark, in.Active, actor).Scan(&id)
	return id, mapWriteError(err, "Zone", in.Code)
}

func (r *MasterRepository) UpdateZone(ctx context.Context, tx database.Querier, id int, in ZoneInput, actor string) error {
	tag, err := r.q(tx).Exec(ctx, `
		UPDATE zone SET code = $2, name = $3, supervisor_name = $4, remark = $5,
		                active = COALESCE($6, active),
		                version = version + 1, updated_at = now(), updated_by = $7
		 WHERE id = $1 AND version = $8`,
		id, in.Code, in.Name, in.SupervisorName, in.Remark, in.Active, actor, in.Version)
	if err != nil {
		return mapWriteError(err, "Zone", in.Code)
	}
	if tag.RowsAffected() == 0 {
		return r.explainMiss(ctx, tx, "zone", "Zone", id)
	}
	return nil
}

// ---------------------------------------------------------------- blocks

const blockSelect = `
	SELECT b.id, b.code, b.name,
	       z.id, z.code, z.name, f.id, f.code, f.name, p.id, p.name, c.id, c.name,
	       b.total_area_ha, b.plantable_area_ha, b.non_plantable_area_ha,
	       b.land_status::text, b.cane_status::text, b.latitude, b.longitude,
	       b.boundary IS NOT NULL,
	       CASE WHEN b.boundary IS NULL THEN NULL ELSE round((ST_Area(b.boundary) / 10000)::numeric, 4) END,
	       b.remark, b.active, b.version, b.updated_at, b.updated_by
	  FROM block b
	  JOIN zone z       ON z.id = b.zone_id
	  JOIN farm f       ON f.id = z.farm_id
	  JOIN plantation p ON p.id = f.plantation_id
	  JOIN company c    ON c.id = p.company_id`

func scanBlock(row pgx.Row) (domain.Block, error) {
	var b domain.Block
	err := row.Scan(&b.ID, &b.Code, &b.Name,
		&b.ZoneID, &b.ZoneCode, &b.ZoneName, &b.FarmID, &b.FarmCode, &b.FarmName,
		&b.PlantationID, &b.PlantationName, &b.CompanyID, &b.CompanyName,
		&b.Areas.TotalHa, &b.Areas.PlantableHa, &b.Areas.NonPlantableHa,
		&b.LandStatus, &b.CaneStatus, &b.Latitude, &b.Longitude,
		&b.HasBoundary, &b.BoundaryAreaHa,
		&b.Remark, &b.Active, &b.Version, &b.UpdatedAt, &b.UpdatedBy)
	b.MapURL = domain.GoogleMapsURL(b.Latitude, b.Longitude)
	return b, err
}

func (r *MasterRepository) ListBlocks(ctx context.Context, f domain.Filter, page domain.Page) ([]domain.Block, int, error) {
	b := &builder{}
	if f.ZoneID != nil {
		b.eq("b.zone_id", *f.ZoneID)
	}
	if f.FarmID != nil {
		b.eq("z.farm_id", *f.FarmID)
	}
	if f.BlockID != nil {
		b.eq("b.id", *f.BlockID)
	}
	if f.LandStatus != nil {
		b.where = append(b.where, "b.land_status = "+b.add(*f.LandStatus)+"::land_status")
	}
	if f.CaneStatus != nil {
		b.where = append(b.where, "b.cane_status = "+b.add(*f.CaneStatus)+"::cane_status")
	}
	if !f.IncludeInactive {
		b.raw("b.active")
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		p := b.add("%" + strings.ToLower(s) + "%")
		b.raw(fmt.Sprintf("(lower(b.code) LIKE %s OR lower(b.name) LIKE %s)", p, p))
	}

	var total int
	countQuery := `SELECT count(*) FROM block b JOIN zone z ON z.id = b.zone_id` + b.whereSQL()
	if err := r.db.Pool().QueryRow(ctx, countQuery, b.args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count blocks: %w", err)
	}

	query := blockSelect + b.whereSQL() + fmt.Sprintf(" ORDER BY b.code LIMIT %s OFFSET %s",
		b.add(page.Size), b.add(page.Offset()))
	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list blocks: %w", err)
	}
	defer rows.Close()

	blocks := []domain.Block{}
	for rows.Next() {
		block, err := scanBlock(rows)
		if err != nil {
			return nil, 0, err
		}
		blocks = append(blocks, block)
	}
	return blocks, total, rows.Err()
}

// GetBlock returns the block with everything hanging off it — the unplantable breakdown and the
// planting records — because the detail panel and the map popup both need all of it at once.
func (r *MasterRepository) GetBlock(ctx context.Context, tx database.Querier, id int) (domain.Block, error) {
	q := r.q(tx)
	block, err := scanBlock(q.QueryRow(ctx, blockSelect+" WHERE b.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Block{}, domain.NotFound("Block", id)
	}
	if err != nil {
		return domain.Block{}, err
	}

	parts, err := r.NonPlantableParts(ctx, tx, id)
	if err != nil {
		return domain.Block{}, err
	}
	block.NonPlantableParts = parts

	plantings, err := r.ListPlantings(ctx, tx, domain.Filter{BlockID: &id})
	if err != nil {
		return domain.Block{}, err
	}
	block.Plantings = plantings

	// The derived figures come from the same helper the validator uses, so a block's detail can
	// never disagree with what the dashboard reports for it.
	var newHa, ratoonHa float64
	for _, p := range plantings {
		if p.PlantingType == "Ratoon" {
			ratoonHa += p.ActualAreaHa
		} else {
			newHa += p.ActualAreaHa
		}
	}
	block.Areas = domain.DeriveBlockAreas(domain.BlockAreaInput{
		TotalHa: block.Areas.TotalHa, PlantableHa: block.Areas.PlantableHa,
		NewPlantingHa: newHa, RatoonHa: ratoonHa,
	})
	return block, nil
}

type BlockInput struct {
	ZoneID          int      `json:"zoneId"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	TotalAreaHa     float64  `json:"totalAreaHa"`
	PlantableAreaHa float64  `json:"plantableAreaHa"`
	LandStatus      string   `json:"landStatus"`
	CaneStatus      string   `json:"caneStatus"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	Remark          *string  `json:"remark"`
	Active          *bool    `json:"active"`
	Version         int      `json:"version"`
}

func (r *MasterRepository) CreateBlock(ctx context.Context, tx database.Querier, in BlockInput, actor string) (int, error) {
	var id int
	err := r.q(tx).QueryRow(ctx, `
		INSERT INTO block(zone_id, code, name, total_area_ha, plantable_area_ha,
		                  land_status, cane_status, latitude, longitude, remark, active, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6::land_status, $7::cane_status, $8, $9, $10, COALESCE($11, true), $12, $12)
		RETURNING id`,
		in.ZoneID, in.Code, in.Name, in.TotalAreaHa, in.PlantableAreaHa,
		in.LandStatus, in.CaneStatus, in.Latitude, in.Longitude, in.Remark, in.Active, actor).Scan(&id)
	return id, mapWriteError(err, "Block", in.Code)
}

func (r *MasterRepository) UpdateBlock(ctx context.Context, tx database.Querier, id int, in BlockInput, actor string) error {
	tag, err := r.q(tx).Exec(ctx, `
		UPDATE block SET code = $2, name = $3, total_area_ha = $4, plantable_area_ha = $5,
		                 land_status = $6::land_status, cane_status = $7::cane_status,
		                 latitude = $8, longitude = $9, remark = $10, active = COALESCE($11, active),
		                 version = version + 1, updated_at = now(), updated_by = $12
		 WHERE id = $1 AND version = $13`,
		id, in.Code, in.Name, in.TotalAreaHa, in.PlantableAreaHa, in.LandStatus, in.CaneStatus,
		in.Latitude, in.Longitude, in.Remark, in.Active, actor, in.Version)
	if err != nil {
		return mapWriteError(err, "Block", in.Code)
	}
	if tag.RowsAffected() == 0 {
		return r.explainMiss(ctx, tx, "block", "Block", id)
	}
	return nil
}

// ---------------------------------------------------------------- geometry

// Geometry returns the block's boundary as GeoJSON along with the area PostGIS measures for it.
func (r *MasterRepository) Geometry(ctx context.Context, id int) (json.RawMessage, *float64, error) {
	var geo *string
	var areaHa *float64
	err := r.db.Pool().QueryRow(ctx, `
		SELECT ST_AsGeoJSON(boundary::geometry),
		       CASE WHEN boundary IS NULL THEN NULL ELSE round((ST_Area(boundary) / 10000)::numeric, 4) END
		  FROM block WHERE id = $1`, id).Scan(&geo, &areaHa)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, domain.NotFound("Block", id)
	}
	if err != nil {
		return nil, nil, err
	}
	if geo == nil {
		return nil, nil, nil
	}
	return json.RawMessage(*geo), areaHa, nil
}

// SetGeometry replaces a block's boundary from GeoJSON and returns the area PostGIS measures.
// ST_Multi accepts a single Polygon as well as a MultiPolygon, so a caller does not have to know
// which the column holds.
func (r *MasterRepository) SetGeometry(ctx context.Context, tx database.Querier, id int, geoJSON json.RawMessage, actor string) (float64, error) {
	var areaHa float64
	err := r.q(tx).QueryRow(ctx, `
		UPDATE block
		   SET boundary = ST_Multi(ST_SetSRID(ST_GeomFromGeoJSON($2), 4326))::geography,
		       version = version + 1, updated_at = now(), updated_by = $3
		 WHERE id = $1
		RETURNING round((ST_Area(boundary) / 10000)::numeric, 4)`,
		id, string(geoJSON), actor).Scan(&areaHa)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.NotFound("Block", id)
	}
	if err != nil {
		return 0, domain.BadRequest("INVALID_GEOMETRY",
			"The boundary is not valid GeoJSON that PostGIS can store as a polygon: "+err.Error())
	}
	return areaHa, nil
}

// ---------------------------------------------------------------- non-plantable breakdown

func (r *MasterRepository) NonPlantableParts(ctx context.Context, tx database.Querier, blockID int) ([]domain.NonPlantablePart, error) {
	rows, err := r.q(tx).Query(ctx, `
		SELECT p.reason_code, n.name, p.area_ha, p.remark
		  FROM block_non_plantable p JOIN non_plantable_reason n ON n.code = p.reason_code
		 WHERE p.block_id = $1 ORDER BY n.sort_order`, blockID)
	if err != nil {
		return nil, fmt.Errorf("non-plantable parts: %w", err)
	}
	defer rows.Close()

	parts := []domain.NonPlantablePart{}
	for rows.Next() {
		var p domain.NonPlantablePart
		if err := rows.Scan(&p.ReasonCode, &p.ReasonName, &p.AreaHa, &p.Remark); err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	return parts, rows.Err()
}

// ReplaceNonPlantableParts swaps the whole breakdown for a block in one transaction. Replacing
// rather than merging is what lets the deferred trigger check the finished state once, instead of
// rejecting an intermediate state that the caller was about to fix with the next statement.
func (r *MasterRepository) ReplaceNonPlantableParts(ctx context.Context, tx database.Querier, blockID int, parts []domain.NonPlantablePart) error {
	if _, err := tx.Exec(ctx, `DELETE FROM block_non_plantable WHERE block_id = $1`, blockID); err != nil {
		return err
	}
	for _, p := range parts {
		if p.AreaHa <= 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO block_non_plantable(block_id, reason_code, area_ha, remark)
			VALUES ($1, $2, $3, $4)`, blockID, p.ReasonCode, p.AreaHa, p.Remark); err != nil {
			return mapWriteError(err, "Non-plantable reason", p.ReasonCode)
		}
	}
	return nil
}

// ---------------------------------------------------------------- planting

func (r *MasterRepository) ListPlantings(ctx context.Context, tx database.Querier, f domain.Filter) ([]domain.Planting, error) {
	b := &builder{}
	if f.BlockID != nil {
		b.eq("bp.block_id", *f.BlockID)
	}
	if f.ZoneID != nil {
		b.eq("bl.zone_id", *f.ZoneID)
	}
	if f.FarmID != nil {
		b.eq("z.farm_id", *f.FarmID)
	}
	if f.CropYear != nil {
		b.eq("bp.crop_year", *f.CropYear)
	}
	if f.PlantingYear != nil {
		b.eq("bp.planting_year", *f.PlantingYear)
	}
	if f.SeasonID != nil {
		b.eq("bp.crop_season_id", *f.SeasonID)
	}
	if f.VarietyID != nil {
		b.eq("bp.cane_variety_id", *f.VarietyID)
	}
	if f.PlantingType != nil {
		b.where = append(b.where, "bp.planting_type = "+b.add(*f.PlantingType)+"::planting_type")
	}

	query := `
		SELECT bp.id, bp.block_id, bl.code, bp.crop_season_id, cs.name, bp.crop_year, bp.planting_year,
		       bp.planting_type::text, bp.ratoon_no, bp.cane_variety_id, cv.name,
		       bp.planned_area_ha, bp.actual_area_ha,
		       bp.planned_date::text, bp.actual_date::text, bp.expected_harvest_date::text,
		       bp.remark, bp.version
		  FROM block_planting bp
		  JOIN block bl        ON bl.id = bp.block_id
		  JOIN zone z          ON z.id = bl.zone_id
		  JOIN crop_season cs  ON cs.id = bp.crop_season_id
		  LEFT JOIN cane_variety cv ON cv.id = bp.cane_variety_id` +
		b.whereSQL() + ` ORDER BY bl.code, bp.crop_year, bp.planting_type`

	rows, err := r.q(tx).Query(ctx, query, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list plantings: %w", err)
	}
	defer rows.Close()

	out := []domain.Planting{}
	for rows.Next() {
		var p domain.Planting
		if err := rows.Scan(&p.ID, &p.BlockID, &p.BlockCode, &p.CropSeasonID, &p.CropSeasonName,
			&p.CropYear, &p.PlantingYear, &p.PlantingType, &p.RatoonNo, &p.CaneVarietyID, &p.CaneVarietyName,
			&p.PlannedAreaHa, &p.ActualAreaHa, &p.PlannedDate, &p.ActualDate, &p.ExpectedHarvestDate,
			&p.Remark, &p.Version); err != nil {
			return nil, err
		}
		p.VarianceAreaHa = round4(p.ActualAreaHa - p.PlannedAreaHa)
		p.AchievementPercent = domain.Achievement(p.ActualAreaHa, p.PlannedAreaHa)
		out = append(out, p)
	}
	return out, rows.Err()
}

type PlantingInput struct {
	BlockID             int     `json:"blockId"`
	CropSeasonID        int     `json:"cropSeasonId"`
	CropYear            int     `json:"cropYear"`
	PlantingYear        int     `json:"plantingYear"`
	PlantingType        string  `json:"plantingType"`
	RatoonNo            int     `json:"ratoonNo"`
	CaneVarietyID       *int    `json:"caneVarietyId"`
	PlannedAreaHa       float64 `json:"plannedAreaHa"`
	ActualAreaHa        float64 `json:"actualAreaHa"`
	PlannedDate         *string `json:"plannedDate"`
	ActualDate          *string `json:"actualDate"`
	ExpectedHarvestDate *string `json:"expectedHarvestDate"`
	Remark              *string `json:"remark"`
	Version             int     `json:"version"`
}

func (r *MasterRepository) UpsertPlanting(ctx context.Context, tx database.Querier, in PlantingInput, actor string) (int, error) {
	var id int
	err := tx.QueryRow(ctx, `
		INSERT INTO block_planting(block_id, crop_season_id, crop_year, planting_year, planting_type,
		                           ratoon_no, cane_variety_id, planned_area_ha, actual_area_ha,
		                           planned_date, actual_date, expected_harvest_date, remark,
		                           created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5::planting_type, $6, $7, $8, $9,
		        $10::date, $11::date, $12::date, $13, $14, $14)
		ON CONFLICT (block_id, crop_year, planting_type) DO UPDATE
		   SET crop_season_id = EXCLUDED.crop_season_id,
		       planting_year = EXCLUDED.planting_year,
		       ratoon_no = EXCLUDED.ratoon_no,
		       cane_variety_id = EXCLUDED.cane_variety_id,
		       planned_area_ha = EXCLUDED.planned_area_ha,
		       actual_area_ha = EXCLUDED.actual_area_ha,
		       planned_date = EXCLUDED.planned_date,
		       actual_date = EXCLUDED.actual_date,
		       expected_harvest_date = EXCLUDED.expected_harvest_date,
		       remark = EXCLUDED.remark,
		       version = block_planting.version + 1,
		       updated_at = now(), updated_by = EXCLUDED.updated_by
		RETURNING id`,
		in.BlockID, in.CropSeasonID, in.CropYear, in.PlantingYear, in.PlantingType, in.RatoonNo,
		in.CaneVarietyID, in.PlannedAreaHa, in.ActualAreaHa,
		in.PlannedDate, in.ActualDate, in.ExpectedHarvestDate, in.Remark, actor).Scan(&id)
	return id, mapWriteError(err, "Planting record", fmt.Sprintf("block %d / %d", in.BlockID, in.CropYear))
}

func (r *MasterRepository) DeletePlanting(ctx context.Context, tx database.Querier, id int) error {
	tag, err := tx.Exec(ctx, `DELETE FROM block_planting WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NotFound("Planting record", id)
	}
	return nil
}

// ---------------------------------------------------------------- helpers

// explainMiss tells a caller whose UPDATE matched nothing which of the two possible reasons it
// was. Answering "not found" for a concurrency loss would send them looking for a deleted row.
func (r *MasterRepository) explainMiss(ctx context.Context, tx database.Querier, table, entity string, id int) error {
	var exists bool
	if err := r.q(tx).QueryRow(ctx,
		fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)", table), id).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return domain.StaleVersion(entity, id)
	}
	return domain.NotFound(entity, id)
}

// mapWriteError turns the constraint names and the RAISE messages from the schema into errors a
// user can act on. Anything unrecognised is left alone so it surfaces as a 500 and gets logged.
func mapWriteError(err error, entity, key string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "AREA_EXCEEDS_PLANTABLE"):
		return domain.Invalid([]domain.FieldError{{
			Field: "areaWithCaneHa", Message: afterMarker(msg, "AREA_EXCEEDS_PLANTABLE: ")}})
	case strings.Contains(msg, "AREA_EXCEEDS_NON_PLANTABLE"):
		return domain.Invalid([]domain.FieldError{{
			Field: "nonPlantableParts", Message: afterMarker(msg, "AREA_EXCEEDS_NON_PLANTABLE: ")}})
	case strings.Contains(msg, "block_plantable_within_total"):
		return domain.Invalid([]domain.FieldError{{
			Field: "plantableAreaHa", Message: "Plantable area cannot exceed the total area."}})
	case strings.Contains(msg, "_not_negative"):
		return domain.Invalid([]domain.FieldError{{
			Field: "totalAreaHa", Message: "Area figures cannot be negative."}})
	case strings.Contains(msg, "duplicate key"):
		return domain.Conflict("DUPLICATE_CODE", fmt.Sprintf("%s %q already exists.", entity, key))
	case strings.Contains(msg, "violates foreign key"):
		return domain.BadRequest("INVALID_REFERENCE",
			fmt.Sprintf("%s refers to something that does not exist.", entity))
	}
	return err
}

func afterMarker(msg, marker string) string {
	if i := strings.Index(msg, marker); i >= 0 {
		rest := msg[i+len(marker):]
		if j := strings.Index(rest, " (SQLSTATE"); j >= 0 {
			rest = rest[:j]
		}
		return strings.TrimSpace(rest)
	}
	return msg
}
