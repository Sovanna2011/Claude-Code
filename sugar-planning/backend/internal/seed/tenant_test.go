package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/seed"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
	"github.com/kss/sugarplan/internal/store/memory"
)

// The second tenant. It exists so that data scope is a claim two real factories
// can be held to; these tests hold the services to it, and the acceptance
// harness holds a running instance to it over HTTP.
//
// The reads go through the service and not the store on purpose. The scope is
// applied in the service layer, so that a repository can never be the thing
// that leaks another factory's plan - which means a test that reads the store
// directly is testing the wrong thing, and would fail here whatever the system
// did.

func loadBothTenants(t *testing.T) (store.Store, *service.Planning, seed.DemoResult, seed.TenantResult) {
	t.Helper()
	s := memory.New()
	planning := service.NewPlanning(s, func() time.Time { return time.Now().UTC() })
	analytics := service.NewAnalytics(s, planning)

	first, err := seed.LoadDemo(context.Background(), s, planning, analytics, 14)
	if err != nil {
		t.Fatalf("load the reference scenario: %v", err)
	}
	second, err := seed.LoadSecondTenant(context.Background(), s, planning)
	if err != nil {
		t.Fatalf("load the second tenant: %v", err)
	}
	return s, planning, first, second
}

// planner is an account of one factory and nothing else, which is what the test
// profile gives every development account once there is more than one mill.
func planner(companyID, factoryID string) context.Context {
	return auth.WithPrincipal(context.Background(),
		auth.NewPrincipal("t", "t", "t", "", []string{auth.RoleProductionPlanner},
			[]string{companyID}, []string{factoryID}))
}

func TestTheSecondTenantHasAPlanOfItsOwn(t *testing.T) {
	_, _, _, second := loadBothTenants(t)

	if second.CompanyID == "" || second.FactoryID == "" || second.SeasonID == "" {
		t.Fatal("the second tenant was not created")
	}
	// 640,000 t over 96 days, not 2,300,000 over 137. The figures differ on
	// purpose: a second tenant carrying the same numbers would let a scope leak
	// pass unnoticed, because the test would be reading the right total from
	// the wrong factory.
	if !second.Generated.Summary.CaneAllocated.Equal(domain.D("640000")) {
		t.Errorf("the second tenant's plan allocates %s t of cane, want 640,000",
			second.Generated.Summary.CaneAllocated)
	}
	if second.Generated.Summary.WorkingDays != 96 {
		t.Errorf("the second tenant's season covers %d days, want 96",
			second.Generated.Summary.WorkingDays)
	}
}

func TestNeitherTenantCanSeeTheOther(t *testing.T) {
	s, planning, first, second := loadBothTenants(t)

	theirs := planner(second.CompanyID, second.FactoryID)
	ours := planner(first.CompanyID, first.FactoryID)

	// Seasons.
	page, err := planning.ListSeasons(theirs, store.ListOptions{Top: 100})
	if err != nil {
		t.Fatalf("the second tenant reading its seasons: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != second.SeasonID {
		t.Errorf("the second tenant sees %d seasons; it has exactly one of its own",
			len(page.Items))
	}
	page, err = planning.ListSeasons(ours, store.ListOptions{Top: 100})
	if err != nil {
		t.Fatalf("the first tenant reading its seasons: %v", err)
	}
	for _, season := range page.Items {
		if season.ID == second.SeasonID {
			t.Error("the first tenant can see the second tenant's season")
		}
	}

	// Naming the other tenant's season by id gets no further than listing it.
	if _, err := planning.GetSeason(theirs, first.SeasonID); err == nil {
		t.Error("the second tenant read the first tenant's season by id")
	}
	if _, err := planning.GetSeason(ours, second.SeasonID); err == nil {
		t.Error("the first tenant read the second tenant's season by id")
	}

	// Warehouses. The stores are the other half of a tenant: a plan nobody can
	// read is little use if the stock behind it is common.
	for code, id := range second.Warehouses {
		w, err := s.MasterData().Warehouses().Get(theirs, id)
		if err != nil {
			t.Errorf("the second tenant cannot read its own warehouse %s: %v", code, err)
			continue
		}
		if w.FactoryID != second.FactoryID {
			t.Errorf("warehouse %s belongs to %s, not to the second tenant", code, w.FactoryID)
		}
	}
}

func TestASecondBootDoesNotDoubleTheSecondTenant(t *testing.T) {
	s, planning, _, second := loadBothTenants(t)

	again, err := seed.LoadSecondTenant(context.Background(), s, planning)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if again.SeasonID != second.SeasonID || again.FactoryID != second.FactoryID {
		t.Fatal("the second load created a new tenant rather than finding the one it made")
	}

	ctx := planner(second.CompanyID, second.FactoryID)
	seasons, err := planning.ListSeasons(ctx, store.ListOptions{Top: 100})
	if err != nil {
		t.Fatalf("read the seasons: %v", err)
	}
	if len(seasons.Items) != 1 {
		t.Errorf("%d seasons after two boots, want one", len(seasons.Items))
	}
	// And the plan is the same size, not twice the size: the mix is upserted on
	// its business key, so regenerating writes over the rows rather than adding
	// a second set of them.
	if !again.Generated.Summary.CaneAllocated.Equal(second.Generated.Summary.CaneAllocated) {
		t.Errorf("the plan allocates %s t after two boots, it allocated %s after one",
			again.Generated.Summary.CaneAllocated, second.Generated.Summary.CaneAllocated)
	}
}
