package domain_test

import (
	"net/http"
	"testing"

	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
)

func planner() domain.User { return domain.User{Username: "planner", Role: domain.RolePlanner} }
func manager() domain.User { return domain.User{Username: "manager", Role: domain.RoleManager} }
func viewer() domain.User  { return domain.User{Username: "viewer", Role: domain.RoleViewer} }

func TestCheckTransitionAllowsTheHappyPath(t *testing.T) {
	steps := []struct {
		action string
		from   string
		want   string
		by     domain.User
	}{
		{domain.ActionSubmit, domain.ProjectionDraft, domain.ProjectionSubmitted, planner()},
		{domain.ActionReview, domain.ProjectionSubmitted, domain.ProjectionUnderReview, manager()},
		{domain.ActionApprove, domain.ProjectionUnderReview, domain.ProjectionApproved, manager()},
		{domain.ActionClose, domain.ProjectionApproved, domain.ProjectionClosed, manager()},
	}
	for _, s := range steps {
		got, err := domain.CheckTransition(s.action, s.from, s.by)
		if err != nil {
			t.Fatalf("%s from %s: unexpected refusal: %v", s.action, s.from, err)
		}
		if got.To != s.want {
			t.Fatalf("%s from %s: landed in %s, want %s", s.action, s.from, got.To, s.want)
		}
	}
}

// Approving straight from Submitted skips the review step. That is deliberate: a manager who has
// read the plan should not have to click twice.
func TestApproveSkipsTheReviewStep(t *testing.T) {
	if _, err := domain.CheckTransition(domain.ActionApprove, domain.ProjectionSubmitted, manager()); err != nil {
		t.Fatalf("approving a submitted plan should be allowed: %v", err)
	}
}

func TestCheckTransitionRefusesTheWrongStatus(t *testing.T) {
	// A draft has not been submitted, so there is nothing to approve yet.
	_, err := domain.CheckTransition(domain.ActionApprove, domain.ProjectionDraft, manager())
	assertStatus(t, err, http.StatusConflict, "INVALID_TRANSITION")

	// A closed plan is finished; nothing moves it.
	for _, action := range []string{domain.ActionSubmit, domain.ActionApprove, domain.ActionRevise} {
		_, err := domain.CheckTransition(action, domain.ProjectionClosed, manager())
		assertStatus(t, err, http.StatusConflict, "INVALID_TRANSITION")
	}
}

func TestCheckTransitionRefusesTheWrongRole(t *testing.T) {
	// A planner may submit their own work but may not approve it.
	if _, err := domain.CheckTransition(domain.ActionSubmit, domain.ProjectionDraft, planner()); err != nil {
		t.Fatalf("a planner should be able to submit: %v", err)
	}
	_, err := domain.CheckTransition(domain.ActionApprove, domain.ProjectionSubmitted, planner())
	assertStatus(t, err, http.StatusForbidden, "FORBIDDEN")

	// A report viewer takes no part in the workflow at all.
	_, err = domain.CheckTransition(domain.ActionSubmit, domain.ProjectionDraft, viewer())
	assertStatus(t, err, http.StatusForbidden, "FORBIDDEN")
}

func TestCheckTransitionRefusesAnUnknownAction(t *testing.T) {
	_, err := domain.CheckTransition("Bless", domain.ProjectionDraft, manager())
	assertStatus(t, err, http.StatusBadRequest, "UNKNOWN_ACTION")
}

// The three refusals have to be distinguishable: a client retries a 409, fixes a 400 and gives up
// on a 403. Collapsing them into one status would make all three look like the same problem.
func TestTheThreeRefusalsAreDistinct(t *testing.T) {
	seen := map[int]bool{}
	for _, err := range []error{
		mustFail(t, "Bless", domain.ProjectionDraft, manager()),
		mustFail(t, domain.ActionApprove, domain.ProjectionDraft, manager()),
		mustFail(t, domain.ActionApprove, domain.ProjectionSubmitted, planner()),
	} {
		e, _ := domain.AsError(err)
		seen[e.Status] = true
	}
	if len(seen) != 3 {
		t.Fatalf("the three refusals collapsed onto %d statuses: %v", len(seen), seen)
	}
}

func TestRejectingAndRevisingNeedAReason(t *testing.T) {
	needs := map[string]bool{
		domain.ActionReject:              true,
		domain.ActionReturnForCorrection: true,
		domain.ActionRevise:              true,
		domain.ActionApprove:             false,
		domain.ActionSubmit:              false,
		domain.ActionReview:              false,
		domain.ActionClose:               false,
	}
	for action, want := range needs {
		transition, ok := domain.FindTransition(action)
		if !ok {
			t.Fatalf("the workflow has no action called %q", action)
		}
		if transition.NeedsReason != want {
			t.Fatalf("%s: needs a reason = %v, want %v", action, transition.NeedsReason, want)
		}
	}
	if len(needs) != len(domain.ProjectionWorkflow) {
		t.Fatalf("the workflow has %d actions but this test checks %d",
			len(domain.ProjectionWorkflow), len(needs))
	}
}

func TestAllowedActionsMatchesWhatCheckTransitionPermits(t *testing.T) {
	// The buttons a screen draws come from AllowedActions and the server enforces CheckTransition.
	// If the two ever disagree, a user sees a button that fails or misses one that would work.
	for _, status := range domain.ValidProjectionStatuses {
		for _, u := range []domain.User{planner(), manager(), viewer()} {
			offered := map[string]bool{}
			for _, a := range domain.AllowedActions(status, u) {
				offered[a] = true
			}
			for _, transition := range domain.ProjectionWorkflow {
				_, err := domain.CheckTransition(transition.Action, status, u)
				if (err == nil) != offered[transition.Action] {
					t.Fatalf("%s as %s in %s: offered=%v but CheckTransition says %v",
						transition.Action, u.Role, status, offered[transition.Action], err)
				}
			}
		}
	}
}

func TestLinesAreEditableOnlyWhileDrafting(t *testing.T) {
	if !domain.LinesEditable(domain.ProjectionDraft) {
		t.Fatal("a draft must be editable")
	}
	for _, status := range []string{domain.ProjectionSubmitted, domain.ProjectionUnderReview,
		domain.ProjectionApproved, domain.ProjectionRejected, domain.ProjectionRevised, domain.ProjectionClosed} {
		if domain.LinesEditable(status) {
			t.Fatalf("%s must not be editable", status)
		}
	}
}

// Every status the workflow can land in has to be one the vocabulary knows, or a transition would
// write a value the enum column rejects.
func TestEveryTransitionLandsOnAKnownStatus(t *testing.T) {
	for _, transition := range domain.ProjectionWorkflow {
		if !domain.IsOneOf(transition.To, domain.ValidProjectionStatuses) {
			t.Fatalf("%s lands on unknown status %q", transition.Action, transition.To)
		}
		for _, from := range transition.From {
			if !domain.IsOneOf(from, domain.ValidProjectionStatuses) {
				t.Fatalf("%s starts from unknown status %q", transition.Action, from)
			}
		}
	}
}

func mustFail(t *testing.T, action, status string, u domain.User) error {
	t.Helper()
	_, err := domain.CheckTransition(action, status, u)
	if err == nil {
		t.Fatalf("%s from %s as %s should have been refused", action, status, u.Role)
	}
	return err
}

func assertStatus(t *testing.T, err error, status int, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a %s refusal, got none", code)
	}
	e, ok := domain.AsError(err)
	if !ok {
		t.Fatalf("expected a domain error, got %T: %v", err, err)
	}
	if e.Status != status || e.Code != code {
		t.Fatalf("expected %d/%s, got %d/%s", status, code, e.Status, e.Code)
	}
}
