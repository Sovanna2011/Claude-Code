package auth_test

import (
	"testing"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
)

// The scheduler is not a person, and what that means for its data scope is worth
// pinning down: it sees every factory, and still only what its own roles allow.
func TestTheSchedulerSeesEveryFactoryAndNoMorePermissions(t *testing.T) {
	job := auth.NewSystemPrincipal(auth.RoleIntegration)

	if !job.IsSystem() {
		t.Error("a system principal must say so")
	}
	// A season nobody is signed in to is exactly what a scheduled job is for.
	if err := job.RequireFactory("any-factory-at-all"); err != nil {
		t.Errorf("a job must see every factory: %v", err)
	}
	if !job.CanSeeCompany("any-company") {
		t.Error("a job must see every company")
	}

	// "Runs as the system" must not mean "runs as an administrator".
	if job.Can(domain.PermAdmin) || job.Can(domain.PermPlanWrite) {
		t.Error("a job holds only the permissions of the roles it was given")
	}
	if !job.Can(domain.PermIntegrationWrite) {
		t.Error("and it does hold those")
	}

	// An ordinary principal is unaffected: the widening is not a default.
	person := auth.NewPrincipal("sub", "planner", "Planner", "",
		[]string{auth.RoleProductionPlanner}, nil, []string{"factory-1"})
	if err := person.RequireFactory("factory-2"); err == nil {
		t.Error("a person's data scope must still fail closed")
	}
	if person.IsSystem() {
		t.Error("only NewSystemPrincipal makes a system principal")
	}
}
