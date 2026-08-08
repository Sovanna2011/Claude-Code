package storetest

import (
	"context"
	"testing"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// The cane supply repository, held to the same behaviour in both stores.
//
// This suite exists because the two disagreed. The SQL store returned the
// delivery schedule ordered by date and the in-memory one by source, which is
// the same rows in a different list - and the seed, reading "the first fourteen
// days", got a different fortnight depending on which store it was running
// against. Ordering is part of the contract when a caller takes a prefix of
// the result, so it is asserted here rather than left to each implementation.
func testCaneSupply(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)

	md := s.MasterData()
	var sources []domain.CaneSource
	for _, c := range []struct {
		code, name string
		kind       domain.SourceType
		hectares   string
		yield      string
	}{
		{"CS-A", "Estate block A", domain.SourceEstate, "100", "70"},
		{"CS-B", "Contract farms B", domain.SourceContract, "200", "65"},
	} {
		saved, err := md.CaneSources().Save(ctx, domain.CaneSource{
			FactoryID: f.factory, Code: c.code, Name: c.name, Type: c.kind,
			Zone: "Test zone", Hectares: domain.D(c.hectares),
			ExpectedYieldTPH: domain.D(c.yield), ExpectedPolPct: domain.D("12.5"),
			TruckCapacityTons: domain.D("18"), TrucksPerDay: 40,
			Validity: domain.Validity{Active: true},
		}, "seed")
		must(t, err, "save cane source "+c.code)
		sources = append(sources, saved)
	}

	// A source is master data and resolves by its business key, like every
	// other master entity.
	got, err := md.CaneSources().GetByCode(ctx, "CS-A")
	must(t, err, "get cane source by code")
	if got.ID != sources[0].ID || got.Type != domain.SourceEstate {
		t.Errorf("GetByCode returned %+v", got)
	}

	// --- commitments --------------------------------------------------------
	season, err := s.Planning().SaveSeason(ctx, domain.Season{
		CompanyID: f.company, FactoryID: f.factory, Code: "SUPPLY-1",
		Name: "Supply test season", StartDate: "2026-12-01", PlannedDays: 30,
		Status: "OPEN",
	}, "seed")
	must(t, err, "save season")
	version, err := s.Planning().SaveVersion(ctx, domain.PlanVersion{
		SeasonID: season.ID, Code: "V1", PlanType: domain.PlanTypeBudget,
		Status: domain.StatusDraft, EffectiveFrom: "2026-12-01",
	}, "seed")
	must(t, err, "save version")

	for i, src := range sources {
		_, err := s.Planning().SaveSupply(ctx, domain.CaneSupplyEntry{
			VersionID: version.ID, SourceID: src.ID,
			HarvestFrom:   domain.BusinessDate("2026-12-01").AddDays(i * 5),
			HarvestTo:     domain.BusinessDate("2026-12-10").AddDays(i * 5),
			CommittedTons: domain.D("7000"),
		}, "seed")
		must(t, err, "save supply commitment")
	}

	entries, err := s.Planning().ListSupply(ctx, version.ID)
	must(t, err, "list supply")
	if len(entries) != 2 {
		t.Fatalf("supply entries = %d, want 2", len(entries))
	}

	// One commitment per source per version: saving the same pair again is a
	// correction, not a second contract.
	updated, err := s.Planning().SaveSupply(ctx, domain.CaneSupplyEntry{
		VersionID: version.ID, SourceID: sources[0].ID,
		HarvestFrom: "2026-12-01", HarvestTo: "2026-12-10",
		CommittedTons: domain.D("8000"),
	}, "seed")
	must(t, err, "re-save supply commitment")
	entries, err = s.Planning().ListSupply(ctx, version.ID)
	must(t, err, "list supply again")
	if len(entries) != 2 {
		t.Errorf("re-saving a source's commitment made %d entries, want 2", len(entries))
	}
	if !updated.CommittedTons.Equal(domain.D("8000")) {
		t.Errorf("the correction stored %s, want 8000", updated.CommittedTons)
	}

	// --- the delivery schedule ---------------------------------------------
	var rows []domain.DailyCaneSupply
	for d := 0; d < 3; d++ {
		date := domain.BusinessDate("2026-12-01").AddDays(d)
		// Deliberately inserted source-last-first, so an implementation that
		// returns insertion order rather than sorting is caught.
		for i := len(sources) - 1; i >= 0; i-- {
			rows = append(rows, domain.DailyCaneSupply{
				VersionID: version.ID, SourceID: sources[i].ID, FactoryID: f.factory,
				BusinessDate: date, Series: domain.SeriesPlan,
				Tons: domain.D("500"), Trips: 28, PolPct: domain.D("12.5"),
			})
		}
	}
	n, err := s.Planning().UpsertCaneSupply(ctx, rows, "seed")
	must(t, err, "upsert delivery schedule")
	if n != 6 {
		t.Errorf("wrote %d schedule rows, want 6", n)
	}

	stored, err := s.Planning().ListCaneSupply(ctx, store.PlanFilter{
		VersionIDs: []string{version.ID},
	})
	must(t, err, "list the schedule")
	if len(stored) != 6 {
		t.Fatalf("schedule rows = %d, want 6", len(stored))
	}
	// Date first. A caller taking the first N rows is taking the earliest
	// days, and that has to be true of both stores.
	for i := 1; i < len(stored); i++ {
		if stored[i].BusinessDate < stored[i-1].BusinessDate {
			t.Fatalf("the schedule is not ordered by date: %s came after %s",
				stored[i].BusinessDate, stored[i-1].BusinessDate)
		}
	}

	// The natural key is (version, source, date, series): writing the same row
	// again corrects it rather than duplicating it.
	rows[0].Tons = domain.D("650")
	if _, err := s.Planning().UpsertCaneSupply(ctx, rows[:1], "seed"); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	stored, err = s.Planning().ListCaneSupply(ctx, store.PlanFilter{
		VersionIDs: []string{version.ID},
	})
	must(t, err, "list after re-upsert")
	if len(stored) != 6 {
		t.Errorf("re-writing a row made %d rows, want 6", len(stored))
	}

	// PLAN and ACTUAL are different rows on the same day, so a delivery never
	// overwrites the schedule it was measured against.
	actual := rows[0]
	actual.Series, actual.Tons = domain.SeriesActual, domain.D("480")
	if _, err := s.Planning().UpsertCaneSupply(ctx, []domain.DailyCaneSupply{actual}, "seed"); err != nil {
		t.Fatalf("upsert an actual: %v", err)
	}
	planned, err := s.Planning().ListCaneSupply(ctx, store.PlanFilter{
		VersionIDs: []string{version.ID}, Series: domain.SeriesPlan,
	})
	must(t, err, "list the plan series")
	if len(planned) != 6 {
		t.Errorf("an actual delivery changed the schedule: %d plan rows, want 6", len(planned))
	}

	// Filtering by source is what a grower's own page reads.
	one, err := s.Planning().ListCaneSupply(ctx, store.PlanFilter{
		VersionIDs: []string{version.ID}, SourceIDs: []string{sources[0].ID},
		Series: domain.SeriesPlan,
	})
	must(t, err, "list one source")
	if len(one) != 3 {
		t.Errorf("one source over three days = %d rows, want 3", len(one))
	}
	for _, r := range one {
		if r.SourceID != sources[0].ID {
			t.Errorf("the source filter returned %s", r.SourceID)
		}
	}

	// And a date range narrows it the way every other daily fact does.
	window, err := s.Planning().ListCaneSupply(ctx, store.PlanFilter{
		VersionIDs: []string{version.ID}, Series: domain.SeriesPlan,
		From: "2026-12-02", To: "2026-12-02",
	})
	must(t, err, "list one day")
	if len(window) != 2 {
		t.Errorf("one day across two sources = %d rows, want 2", len(window))
	}

	// Deleting a commitment leaves the schedule alone: the rows are facts that
	// were true when they were written.
	if err := s.Planning().DeleteSupply(ctx, entries[0].ID); err != nil {
		t.Fatalf("delete supply: %v", err)
	}
	remaining, err := s.Planning().ListSupply(ctx, version.ID)
	must(t, err, "list supply after delete")
	if len(remaining) != 1 {
		t.Errorf("supply entries after delete = %d, want 1", len(remaining))
	}
}
