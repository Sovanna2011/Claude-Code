package seed

import (
	"context"
	"fmt"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
)

// The reference workbook, as data.
//
// Source: `ProductionPlan_2627_2.3mt Rev.1 (corrected)`, sheet
// `RW'2627(2.3mt)Rev1(re)`, prepared by the mill's finance manager on 16 July
// 2026. Until it was supplied, the season target was the only thing known about
// the plan and everything under it was reasonable invention. It is not any more,
// and the figures below are read off the sheet rather than assumed.
//
// The single most important thing the workbook says is that a crushing season
// is not a straight line. The mill opens at 17,000 t while the boilers come up,
// settles at 19,000, drops to half rate the day before each wash-out, stops for
// the wash-out itself, and runs down through 15,000, 12,000, 8,000, 4,000 and
// 3,000 t as the last of the cane arrives. Six wash-outs, so 137 days of
// campaign carry 131 days of crushing.
//
// The plan the generator used to produce - 16,788 t every day for 137 days -
// has the same season total and a different answer to every question anybody
// asks it. The raw silo fills on the wrong date, which is the date the jumbo
// bagging is planned around.

// ReferenceCrushingProfile is the 2026/27 campaign shape.
//
// The plateau is deliberately absent: it is solved from the target, so changing
// the season tonnage gives a plan that still looks like a season instead of one
// that quietly keeps the old rate. Against 2,300,000 t over 137 days it solves
// to the workbook's own 19,000 t.
func ReferenceCrushingProfile() domain.CrushingProfile {
	return domain.CrushingProfile{
		// Two days of start-up while steam is raised.
		RampUp: []domain.CrushingStep{{Rate: domain.DI(17000), Days: 2}},

		// The mill washes out roughly every three weeks. The first five fall on
		// a 21-day cadence; the last is pulled forward from day 126 to day 119
		// so it does not land in the run-down - and the sheet's own note says
		// the day "is adjusted according to suitability and situation", so the
		// list has to be able to say what a cadence cannot.
		CleaningDays:    []int{21, 42, 63, 84, 105, 119},
		PreCleaningRate: domain.DI(9000),

		// The cane thins out at the end of the season.
		RunDown: []domain.CrushingStep{
			{Rate: domain.DI(15000), Days: 5},
			{Rate: domain.DI(12000), Days: 3},
			{Rate: domain.DI(8000), Days: 2},
			{Rate: domain.DI(4000), Days: 2},
			{Rate: domain.DI(3000), Days: 2},
		},
	}
}

// Figures the workbook states, used by the seed and asserted by the tests.
//
// They are constants rather than numbers typed twice, because the whole value
// of having the file is that these are the mill's figures and not somebody's
// recollection of them.
const (
	// WorkbookCaneTons and WorkbookCampaignDays: 1 December 2026 to 16 April
	// 2027 inclusive.
	WorkbookCaneTons     = "2300000"
	WorkbookCampaignDays = 137
	WorkbookCrushingDays = 131
	WorkbookCleaningDays = 6
	WorkbookPlateauTons  = "19000"

	// Raw sugar at 11.00 % recovery, and its split. 124,950 t goes straight to
	// the refinery and 128,050 t into the silo - 49.387 % direct, which is the
	// figure the plan was already carrying and which the workbook confirms.
	WorkbookRawSugarTons   = "253000"
	WorkbookDirectTons     = "124950"
	WorkbookToStorageTons  = "128050"
	WorkbookDirectPct      = "49.387"
	WorkbookRefinedTons    = "106700"
	WorkbookWhiteTons      = "133400"
	WorkbookSuperTons      = "2000"
	WorkbookFinishedTons   = "242100"
	WorkbookJumboTons      = "20700"
	WorkbookJumboRateTPD   = "300"
	WorkbookQuotaRateTPD   = "500"
	WorkbookQuotaDays      = 272
	WorkbookQuotaTons      = "136000"
	WorkbookRemainingTons  = "106100"
	WorkbookRawCapacity    = "110000"
	WorkbookFinishCapacity = "69000"

	// Dates the workbook reports from its own daily stock tracker.
	WorkbookSeasonStart      = domain.BusinessDate("2026-12-01")
	WorkbookSeasonEnd        = domain.BusinessDate("2027-04-16")
	WorkbookRemeltEnd        = domain.BusinessDate("2027-09-02")
	WorkbookJumboFrom        = domain.BusinessDate("2027-01-21")
	WorkbookJumboTo          = domain.BusinessDate("2027-03-30")
	WorkbookFinishedFullDate = domain.BusinessDate("2027-05-28")
	WorkbookNoDeliveryFull   = domain.BusinessDate("2027-02-20")
)

// LoadCrushingProfile stores the reference campaign shape against a version, so
// that regenerating the plan reproduces the mill's curve rather than flattening
// it.
func LoadCrushingProfile(ctx context.Context, planning *service.Planning, versionID string) error {
	if _, err := planning.SaveCrushingProfile(ctx, versionID, ReferenceCrushingProfile()); err != nil {
		return fmt.Errorf("store the crushing profile: %w", err)
	}
	return nil
}

// at is a zero-based column position, as a pointer because a mapping
// distinguishes "column 0" from "no column given".
func at(i int) *int { return &i }
