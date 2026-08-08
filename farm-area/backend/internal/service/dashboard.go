package service

import (
	"context"

	"github.com/sovanna2011/farm-area/backend/internal/domain"
	"github.com/sovanna2011/farm-area/backend/internal/repository"
)

// DashboardService assembles what the dashboard shows. Every method takes the same filter, so the
// KPI cards, the charts, the tree and the map are guaranteed to describe the same land.
type DashboardService struct {
	repo *repository.DashboardRepository
}

// KPIResponse is the top of the dashboard: six figures, each with its share of the total area, and
// the block count the whole thing was calculated from.
type KPIResponse struct {
	Areas      domain.Areas `json:"areas"`
	KPIs       []domain.KPI `json:"kpis"`
	BlockCount int          `json:"blockCount"`
	Unit       string       `json:"unit"`
}

func (s *DashboardService) KPIs(ctx context.Context, f domain.Filter) (KPIResponse, error) {
	areas, blocks, err := s.repo.Totals(ctx, f)
	if err != nil {
		return KPIResponse{}, err
	}

	// Every percentage is of the total area, which is what makes the cards comparable: "60% with
	// cane" and "25% available" are shares of the same denominator and can be read side by side.
	pct := func(v float64) float64 { return domain.PercentOfTotal(v, areas.TotalHa) }

	return KPIResponse{
		Areas:      areas,
		BlockCount: blocks,
		Unit:       "ha",
		KPIs: []domain.KPI{
			{Key: "total", Label: "Total Area", AreaHa: areas.TotalHa, PercentTotal: pct(areas.TotalHa), DrillTo: "tree"},
			{Key: "newPlanting", Label: "New Planting", AreaHa: areas.NewPlantingHa, PercentTotal: pct(areas.NewPlantingHa), DrillTo: "tree?plantingType=NewPlanting"},
			{Key: "ratoon", Label: "Ratoon", AreaHa: areas.RatoonHa, PercentTotal: pct(areas.RatoonHa), DrillTo: "tree?plantingType=Ratoon"},
			{Key: "withCane", Label: "Area with Cane", AreaHa: areas.WithCaneHa, PercentTotal: pct(areas.WithCaneHa), DrillTo: "tree?caneStatus=Growing"},
			{Key: "available", Label: "Available for Planting", AreaHa: areas.AvailableHa, PercentTotal: pct(areas.AvailableHa), DrillTo: "tree"},
			{Key: "cannotPlant", Label: "Cannot Be Planted", AreaHa: areas.NonPlantableHa, PercentTotal: pct(areas.NonPlantableHa), DrillTo: "tree"},
		},
	}, nil
}

func (s *DashboardService) Tree(ctx context.Context, f domain.Filter) ([]*domain.TreeNode, error) {
	return s.repo.Tree(ctx, f)
}

func (s *DashboardService) MapData(ctx context.Context, f domain.Filter) (map[string]any, error) {
	blocks, err := s.repo.MapData(ctx, f)
	if err != nil {
		return nil, err
	}
	outlines, err := s.repo.Outlines(ctx, f)
	if err != nil {
		return nil, err
	}
	return map[string]any{"blocks": blocks, "outlines": outlines}, nil
}

// Charts returns every series the analysis section draws, in one request: five round trips for one
// screen is five chances for the charts to disagree with each other.
func (s *DashboardService) Charts(ctx context.Context, f domain.Filter) ([]domain.ChartSeries, error) {
	areas, _, err := s.repo.Totals(ctx, f)
	if err != nil {
		return nil, err
	}

	byFarm, err := s.repo.AreaByLevel(ctx, f, "farm")
	if err != nil {
		return nil, err
	}
	byZone, err := s.repo.AreaByLevel(ctx, f, "zone")
	if err != nil {
		return nil, err
	}
	monthly, err := s.repo.MonthlyProgress(ctx, f)
	if err != nil {
		return nil, err
	}

	return []domain.ChartSeries{
		{
			Key: "landUtilisation", Title: "Land utilisation", Unit: "ha",
			Points: []domain.ChartPoint{
				{Category: "Area with cane", Value: areas.WithCaneHa},
				{Category: "Available for planting", Value: areas.AvailableHa},
				{Category: "Cannot be planted", Value: areas.NonPlantableHa},
			},
		},
		{
			Key: "caneArea", Title: "New planting versus ratoon", Unit: "ha",
			Points: []domain.ChartPoint{
				{Category: "New planting", Value: areas.NewPlantingHa},
				{Category: "Ratoon", Value: areas.RatoonHa},
			},
		},
		{Key: "areaByFarm", Title: "Area by farm", Unit: "ha", Points: byFarm},
		{Key: "areaByZone", Title: "Area by zone", Unit: "ha", Points: byZone},
		{Key: "monthlyProgress", Title: "Planting progress by month", Unit: "ha", Points: monthly},
	}, nil
}

// PlanActualResponse is the bottom of the dashboard: the programme against what happened, both
// grouped by place and broken down by month.
type PlanActualResponse struct {
	Planned     domain.Areas           `json:"plannedAreas"`
	Actual      domain.Areas           `json:"actualAreas"`
	Variance    domain.Areas           `json:"varianceAreas"`
	Achievement float64                `json:"achievementPercent"`
	Level       string                 `json:"level"`
	Rows        []domain.PlanActualRow `json:"rows"`
	Monthly     []domain.PlanActualRow `json:"monthly"`
}

func (s *DashboardService) PlanVsActual(ctx context.Context, f domain.Filter, level string) (PlanActualResponse, error) {
	actual, _, err := s.repo.Totals(ctx, f)
	if err != nil {
		return PlanActualResponse{}, err
	}
	planned, err := s.repo.PlannedTotals(ctx, f)
	if err != nil {
		return PlanActualResponse{}, err
	}
	rows, err := s.repo.PlanVsActual(ctx, f, level)
	if err != nil {
		return PlanActualResponse{}, err
	}
	monthly, err := s.repo.MonthlyPlanVsActual(ctx, f)
	if err != nil {
		return PlanActualResponse{}, err
	}

	// Variance is actual minus planned, figure by figure. A negative available area means more
	// land was planted than the programme called for, which is worth seeing rather than clamping.
	variance := domain.Areas{
		TotalHa:        actual.TotalHa - planned.TotalHa,
		PlantableHa:    actual.PlantableHa - planned.PlantableHa,
		NonPlantableHa: actual.NonPlantableHa - planned.NonPlantableHa,
		NewPlantingHa:  actual.NewPlantingHa - planned.NewPlantingHa,
		RatoonHa:       actual.RatoonHa - planned.RatoonHa,
		WithCaneHa:     actual.WithCaneHa - planned.WithCaneHa,
		AvailableHa:    actual.AvailableHa - planned.AvailableHa,
	}
	variance.Round()

	if len(rows) == 0 {
		rows = []domain.PlanActualRow{}
	}
	if len(monthly) == 0 {
		monthly = []domain.PlanActualRow{}
	}

	return PlanActualResponse{
		Planned:     planned,
		Actual:      actual,
		Variance:    variance,
		Achievement: domain.Achievement(actual.WithCaneHa, planned.WithCaneHa),
		Level:       level,
		Rows:        rows,
		Monthly:     monthly,
	}, nil
}

// ---------------------------------------------------------------- lookups and audit

type SupportService struct {
	repo *repository.SupportRepository
}

func (s *SupportService) Lookup(ctx context.Context, kind string, parentID *int) ([]repository.LookupItem, error) {
	return s.repo.Lookup(ctx, kind, parentID)
}

func (s *SupportService) Audit(ctx context.Context, u domain.User, entity string, page domain.Page) (domain.PagedResult[repository.AuditEntry], error) {
	// The audit trail names who did what; only an administrator sees it.
	if err := requireRole(u, domain.RoleAdmin); err != nil {
		return domain.PagedResult[repository.AuditEntry]{}, err
	}
	page = page.Normalise()
	items, total, err := s.repo.ListAudit(ctx, entity, page)
	if err != nil {
		return domain.PagedResult[repository.AuditEntry]{}, err
	}
	return domain.NewPagedResult(items, page, total), nil
}
