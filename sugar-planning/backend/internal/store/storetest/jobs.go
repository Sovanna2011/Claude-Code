package storetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/domain"
)

// The scheduler's lease is what stops two instances doing the same work twice,
// so both stores have to agree about exactly when a lease may be taken.

func testJobs(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	now := time.Date(2027, 2, 1, 3, 0, 0, 0, time.UTC)
	const job = "outbox-dispatch"

	got, err := s.Jobs().Acquire(ctx, job, "instance-a", time.Minute, now)
	must(t, err, "acquire")
	if !got {
		t.Fatal("a job nobody holds must be acquirable")
	}

	// The second instance woke at the same moment and must not run it too.
	got, err = s.Jobs().Acquire(ctx, job, "instance-b", time.Minute, now)
	must(t, err, "acquire from the other instance")
	if got {
		t.Fatal("two instances must not hold the same lease")
	}
	// Nor may it record an outcome for a run that is not its own.
	if err := s.Jobs().Finish(ctx, job, "instance-b", now, domain.JobOK, "not mine"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("finishing somebody else's run must be refused, got %v", err)
	}

	must(t, s.Jobs().Finish(ctx, job, "instance-a", now.Add(2*time.Second), domain.JobOK,
		"published 3"), "finish")

	runs, err := s.Jobs().List(ctx)
	must(t, err, "list")
	if len(runs) != 1 {
		t.Fatalf("one job has run, got %d", len(runs))
	}
	if runs[0].Status != domain.JobOK || runs[0].Runs != 1 || runs[0].Failures != 0 {
		t.Errorf("the run must be recorded: %+v", runs[0])
	}
	if runs[0].Detail != "published 3" || runs[0].FinishedAt == nil {
		t.Errorf("what the job did and when it finished are the whole point: %+v", runs[0])
	}

	// Once it is finished, the next tick may take it - either instance.
	got, err = s.Jobs().Acquire(ctx, job, "instance-b", time.Minute, now.Add(time.Minute))
	must(t, err, "acquire after finishing")
	if !got {
		t.Fatal("a finished job must be acquirable again")
	}
	must(t, s.Jobs().Finish(ctx, job, "instance-b", now.Add(61*time.Second), domain.JobFailed,
		"the ERP refused"), "finish with a failure")

	runs, err = s.Jobs().List(ctx)
	must(t, err, "list again")
	if runs[0].Runs != 2 || runs[0].Failures != 1 {
		t.Errorf("both runs and the failure must be counted: %+v", runs[0])
	}

	// An instance that died holding the lease must not block the job for ever.
	got, err = s.Jobs().Acquire(ctx, job, "instance-a", time.Minute, now.Add(2*time.Minute))
	must(t, err, "acquire before the crash")
	if !got {
		t.Fatal("acquire")
	}
	crashed := now.Add(2 * time.Minute)
	got, err = s.Jobs().Acquire(ctx, job, "instance-b", time.Minute, crashed.Add(30*time.Second))
	must(t, err, "acquire while the lease is still live")
	if got {
		t.Error("a live lease must be respected even if its holder is gone")
	}
	got, err = s.Jobs().Acquire(ctx, job, "instance-b", time.Minute, crashed.Add(2*time.Minute))
	must(t, err, "acquire after the lease expired")
	if !got {
		t.Error("an expired lease must be taken over; a dead instance cannot block a job for ever")
	}
}
