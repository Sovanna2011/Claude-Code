// Package service holds the business logic. It sits between the HTTP handlers and the
// repositories: handlers do no more than parse and render, repositories do no more than SQL, and
// everything that decides whether a change is allowed happens here, inside a transaction.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sovanna2011/farm-area/backend/internal/database"
	"github.com/sovanna2011/farm-area/backend/internal/domain"
	"github.com/sovanna2011/farm-area/backend/internal/repository"
)

// Container is the composition root's product: one value carrying every service, built once at
// start-up and handed to the router. Constructor injection keeps the dependencies explicit and
// makes each service testable with a different repository.
type Container struct {
	Master    *MasterService
	Dashboard *DashboardService
	Report    *ReportService
	Support   *SupportService
}

func New(db *database.DB, master *repository.MasterRepository, dash *repository.DashboardRepository,
	support *repository.SupportRepository) *Container {
	m := &MasterService{db: db, repo: master, audit: support}
	d := &DashboardService{repo: dash}
	return &Container{
		Master:    m,
		Dashboard: d,
		Report:    &ReportService{dashboard: d},
		Support:   &SupportService{repo: support},
	}
}

// actor is the caller as the services need them: a name for the audit trail and a role for the
// permission checks.
type actor struct {
	User   domain.User
	Remote string
}

// ---------------------------------------------------------------- master data

type MasterService struct {
	db    *database.DB
	repo  *repository.MasterRepository
	audit *repository.SupportRepository
}

func (s *MasterService) ListFarms(ctx context.Context, f domain.Filter, p domain.Page) (domain.PagedResult[domain.Farm], error) {
	p = p.Normalise()
	items, total, err := s.repo.ListFarms(ctx, f, p)
	if err != nil {
		return domain.PagedResult[domain.Farm]{}, err
	}
	return domain.NewPagedResult(items, p, total), nil
}

func (s *MasterService) GetFarm(ctx context.Context, id int) (domain.Farm, error) {
	return s.repo.GetFarm(ctx, nil, id)
}

func (s *MasterService) SaveFarm(ctx context.Context, u domain.User, remote string, id *int, in repository.FarmInput) (domain.Farm, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return domain.Farm{}, err
	}
	if err := requireText("code", in.Code, 20); err != nil {
		return domain.Farm{}, err
	}
	if err := requireText("name", in.Name, 150); err != nil {
		return domain.Farm{}, err
	}

	var saved domain.Farm
	err := s.db.InTx(ctx, func(tx database.Querier) error {
		var before any
		action := "Create"
		newID := 0

		if id == nil {
			created, err := s.repo.CreateFarm(ctx, tx, in, u.Username)
			if err != nil {
				return err
			}
			newID = created
		} else {
			action = "Update"
			newID = *id
			existing, err := s.repo.GetFarm(ctx, tx, newID)
			if err != nil {
				return err
			}
			before = existing
			if err := s.repo.UpdateFarm(ctx, tx, newID, in, u.Username); err != nil {
				return err
			}
		}

		after, err := s.repo.GetFarm(ctx, tx, newID)
		if err != nil {
			return err
		}
		saved = after
		return s.audit.Write(ctx, tx, u.Username, action, "Farm", fmt.Sprint(newID), before, after, remote)
	})
	return saved, err
}

func (s *MasterService) ListZones(ctx context.Context, f domain.Filter, p domain.Page) (domain.PagedResult[domain.Zone], error) {
	p = p.Normalise()
	items, total, err := s.repo.ListZones(ctx, f, p)
	if err != nil {
		return domain.PagedResult[domain.Zone]{}, err
	}
	return domain.NewPagedResult(items, p, total), nil
}

func (s *MasterService) GetZone(ctx context.Context, id int) (domain.Zone, error) {
	return s.repo.GetZone(ctx, nil, id)
}

func (s *MasterService) SaveZone(ctx context.Context, u domain.User, remote string, id *int, in repository.ZoneInput) (domain.Zone, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return domain.Zone{}, err
	}
	if err := requireText("code", in.Code, 20); err != nil {
		return domain.Zone{}, err
	}
	if err := requireText("name", in.Name, 150); err != nil {
		return domain.Zone{}, err
	}

	var saved domain.Zone
	err := s.db.InTx(ctx, func(tx database.Querier) error {
		var before any
		action := "Create"
		newID := 0

		if id == nil {
			created, err := s.repo.CreateZone(ctx, tx, in, u.Username)
			if err != nil {
				return err
			}
			newID = created
		} else {
			action = "Update"
			newID = *id
			existing, err := s.repo.GetZone(ctx, tx, newID)
			if err != nil {
				return err
			}
			before = existing
			if err := s.repo.UpdateZone(ctx, tx, newID, in, u.Username); err != nil {
				return err
			}
		}

		after, err := s.repo.GetZone(ctx, tx, newID)
		if err != nil {
			return err
		}
		saved = after
		return s.audit.Write(ctx, tx, u.Username, action, "Zone", fmt.Sprint(newID), before, after, remote)
	})
	return saved, err
}

func (s *MasterService) ListBlocks(ctx context.Context, f domain.Filter, p domain.Page) (domain.PagedResult[domain.Block], error) {
	p = p.Normalise()
	items, total, err := s.repo.ListBlocks(ctx, f, p)
	if err != nil {
		return domain.PagedResult[domain.Block]{}, err
	}
	return domain.NewPagedResult(items, p, total), nil
}

func (s *MasterService) GetBlock(ctx context.Context, id int) (domain.Block, error) {
	return s.repo.GetBlock(ctx, nil, id)
}

// BlockSaveRequest is a block and, optionally, the breakdown of why part of it cannot be planted.
// They are saved together because the two have to agree, and the check that they do is deferred
// to the commit.
type BlockSaveRequest struct {
	repository.BlockInput
	NonPlantableParts []domain.NonPlantablePart `json:"nonPlantableParts"`
}

func (s *MasterService) SaveBlock(ctx context.Context, u domain.User, remote string, id *int, req BlockSaveRequest) (domain.Block, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return domain.Block{}, err
	}
	if err := requireText("code", req.Code, 20); err != nil {
		return domain.Block{}, err
	}
	if err := requireText("name", req.Name, 150); err != nil {
		return domain.Block{}, err
	}
	if req.LandStatus == "" {
		req.LandStatus = "Active"
	}
	if req.CaneStatus == "" {
		req.CaneStatus = "Fallow"
	}
	if !domain.IsOneOf(req.LandStatus, domain.ValidLandStatuses) {
		return domain.Block{}, domain.BadRequest("INVALID_LAND_STATUS",
			"Land status must be one of "+strings.Join(domain.ValidLandStatuses, ", ")+".")
	}
	if !domain.IsOneOf(req.CaneStatus, domain.ValidCaneStatuses) {
		return domain.Block{}, domain.BadRequest("INVALID_CANE_STATUS",
			"Cane status must be one of "+strings.Join(domain.ValidCaneStatuses, ", ")+".")
	}

	// The area rules are checked here, against the figures the caller sent plus the cane already
	// recorded on the block, so the message names the field. The database checks them again at
	// commit; this pass exists to give a usable answer, not to be the only guard.
	var existingNew, existingRatoon float64
	if id != nil {
		current, err := s.repo.GetBlock(ctx, nil, *id)
		if err != nil {
			return domain.Block{}, err
		}
		existingNew, existingRatoon = current.Areas.NewPlantingHa, current.Areas.RatoonHa
	}

	var claimed float64
	for _, p := range req.NonPlantableParts {
		claimed += p.AreaHa
	}

	if problems := domain.ValidateBlockAreas(domain.BlockAreaInput{
		TotalHa:           req.TotalAreaHa,
		PlantableHa:       req.PlantableAreaHa,
		NewPlantingHa:     existingNew,
		RatoonHa:          existingRatoon,
		NonPlantableClaim: claimed,
	}); len(problems) > 0 {
		return domain.Block{}, domain.Invalid(problems)
	}

	var saved domain.Block
	err := s.db.InTx(ctx, func(tx database.Querier) error {
		var before any
		action := "Create"
		newID := 0

		if id == nil {
			created, err := s.repo.CreateBlock(ctx, tx, req.BlockInput, u.Username)
			if err != nil {
				return err
			}
			newID = created
		} else {
			action = "Update"
			newID = *id
			existing, err := s.repo.GetBlock(ctx, tx, newID)
			if err != nil {
				return err
			}
			before = existing
			if err := s.repo.UpdateBlock(ctx, tx, newID, req.BlockInput, u.Username); err != nil {
				return err
			}
		}

		if req.NonPlantableParts != nil {
			if err := s.repo.ReplaceNonPlantableParts(ctx, tx, newID, req.NonPlantableParts); err != nil {
				return err
			}
		}

		after, err := s.repo.GetBlock(ctx, tx, newID)
		if err != nil {
			return err
		}
		saved = after
		return s.audit.Write(ctx, tx, u.Username, action, "Block", fmt.Sprint(newID), before, after, remote)
	})
	return saved, err
}

func (s *MasterService) Geometry(ctx context.Context, id int) (json.RawMessage, *float64, error) {
	return s.repo.Geometry(ctx, id)
}

func (s *MasterService) SetGeometry(ctx context.Context, u domain.User, remote string, id int, geoJSON json.RawMessage) (float64, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return 0, err
	}
	if len(geoJSON) == 0 {
		return 0, domain.BadRequest("MISSING_GEOMETRY", "Send the boundary as a GeoJSON geometry.")
	}

	var areaHa float64
	err := s.db.InTx(ctx, func(tx database.Querier) error {
		measured, err := s.repo.SetGeometry(ctx, tx, id, geoJSON, u.Username)
		if err != nil {
			return err
		}
		areaHa = measured
		return s.audit.Write(ctx, tx, u.Username, "SetGeometry", "Block", fmt.Sprint(id), nil,
			map[string]any{"boundaryAreaHa": measured}, remote)
	})
	return areaHa, err
}

// ---------------------------------------------------------------- planting

func (s *MasterService) ListPlantings(ctx context.Context, f domain.Filter) ([]domain.Planting, error) {
	return s.repo.ListPlantings(ctx, nil, f)
}

func (s *MasterService) SavePlanting(ctx context.Context, u domain.User, remote string, in repository.PlantingInput) (domain.Planting, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager, domain.RolePlanner); err != nil {
		return domain.Planting{}, err
	}
	canonical, ok := domain.Canonical(in.PlantingType, domain.ValidPlantingTypes)
	if !ok {
		return domain.Planting{}, domain.BadRequest("INVALID_PLANTING_TYPE",
			"Planting type must be NewPlanting or Ratoon.")
	}
	in.PlantingType = canonical
	if canonical == "Ratoon" && in.RatoonNo < 1 {
		in.RatoonNo = 1
	}
	if canonical == "NewPlanting" {
		in.RatoonNo = 0
	}
	if in.PlannedAreaHa < 0 || in.ActualAreaHa < 0 {
		return domain.Planting{}, domain.Invalid([]domain.FieldError{
			{Field: "plannedAreaHa", Message: "Planted areas cannot be negative."}})
	}

	// Check the block can hold it before the write, so the caller gets a field-level message
	// rather than the deferred trigger's exception at commit.
	block, err := s.repo.GetBlock(ctx, nil, in.BlockID)
	if err != nil {
		return domain.Planting{}, err
	}
	other := block.Areas.WithCaneHa
	for _, p := range block.Plantings {
		if p.CropYear == in.CropYear && p.PlantingType == canonical {
			other -= p.ActualAreaHa // this row is being replaced
		}
	}
	if problems := domain.ValidateBlockAreas(domain.BlockAreaInput{
		TotalHa:       block.Areas.TotalHa,
		PlantableHa:   block.Areas.PlantableHa,
		NewPlantingHa: other + in.ActualAreaHa,
	}); len(problems) > 0 {
		return domain.Planting{}, domain.Invalid(problems)
	}

	var saved domain.Planting
	err = s.db.InTx(ctx, func(tx database.Querier) error {
		id, err := s.repo.UpsertPlanting(ctx, tx, in, u.Username)
		if err != nil {
			return err
		}
		rows, err := s.repo.ListPlantings(ctx, tx, domain.Filter{BlockID: &in.BlockID})
		if err != nil {
			return err
		}
		for _, row := range rows {
			if row.ID == id {
				saved = row
			}
		}
		return s.audit.Write(ctx, tx, u.Username, "SavePlanting", "Planting", fmt.Sprint(id), nil, saved, remote)
	})
	return saved, err
}

func (s *MasterService) DeletePlanting(ctx context.Context, u domain.User, remote string, id int) error {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return err
	}
	return s.db.InTx(ctx, func(tx database.Querier) error {
		if err := s.repo.DeletePlanting(ctx, tx, id); err != nil {
			return err
		}
		return s.audit.Write(ctx, tx, u.Username, "DeletePlanting", "Planting", fmt.Sprint(id), nil, nil, remote)
	})
}

// ---------------------------------------------------------------- shared guards

func requireRole(u domain.User, allowed ...string) error {
	if u.HasAnyRole(allowed...) {
		return nil
	}
	return domain.Forbidden(fmt.Sprintf(
		"Your role (%s) may not make this change; it needs one of: %s.", u.Role, strings.Join(allowed, ", ")))
}

func requireText(field, value string, max int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return domain.Invalid([]domain.FieldError{{Field: field, Message: "This field is required."}})
	}
	if len([]rune(trimmed)) > max {
		return domain.Invalid([]domain.FieldError{{
			Field: field, Message: fmt.Sprintf("Keep this to %d characters or fewer.", max)}})
	}
	return nil
}
