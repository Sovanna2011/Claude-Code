package jobs_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/jobs"
	"github.com/kss/sugarplan/internal/store/memory"
)

// quiet is a logger that throws everything away, so a test run is readable.
func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestOnlyOneInstanceRunsAJob(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	now := time.Date(2027, 2, 1, 3, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	var mu sync.Mutex
	var ran []string
	makeJob := func(instance string) jobs.Job {
		return jobs.Job{
			Name: "dispatch", Every: 5 * time.Millisecond, Timeout: time.Second,
			Run: func(context.Context) (string, error) {
				mu.Lock()
				defer mu.Unlock()
				ran = append(ran, instance)
				return "published 1", nil
			},
		}
	}

	a, err := jobs.New(s, "instance-a", quiet(), clock, makeJob("a"))
	if err != nil {
		t.Fatalf("build scheduler a: %v", err)
	}
	b, err := jobs.New(s, "instance-b", quiet(), clock, makeJob("b"))
	if err != nil {
		t.Fatalf("build scheduler b: %v", err)
	}

	a.Start(ctx)
	b.Start(ctx)
	time.Sleep(60 * time.Millisecond)
	a.Stop()
	b.Stop()

	mu.Lock()
	total := len(ran)
	mu.Unlock()
	if total == 0 {
		t.Fatal("the job never ran")
	}

	runs, err := s.Jobs().List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("one job, one record, got %d", len(runs))
	}
	// Every run is accounted for, and none of them overlapped: the lease is
	// released before the next tick can take it, so the count of recorded runs
	// is the count of runs that happened.
	if runs[0].Runs != int64(total) {
		t.Errorf("recorded %d runs but the job ran %d times", runs[0].Runs, total)
	}
	if runs[0].Status != domain.JobOK || runs[0].Detail != "published 1" {
		t.Errorf("the outcome must be recorded: %+v", runs[0])
	}
}

func TestAFailingJobIsRecordedAndKeepsRunning(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	now := time.Date(2027, 2, 1, 3, 0, 0, 0, time.UTC)

	var attempts int
	var mu sync.Mutex
	scheduler, err := jobs.New(s, "instance-a", quiet(),
		func() time.Time { return now },
		jobs.Job{
			Name: "dispatch", Every: 5 * time.Millisecond, Timeout: time.Second,
			Run: func(context.Context) (string, error) {
				mu.Lock()
				defer mu.Unlock()
				attempts++
				return "", errors.New("the ERP refused the document")
			},
		})
	if err != nil {
		t.Fatalf("build scheduler: %v", err)
	}

	scheduler.Start(ctx)
	time.Sleep(40 * time.Millisecond)
	scheduler.Stop()

	mu.Lock()
	got := attempts
	mu.Unlock()
	if got < 2 {
		t.Fatalf("a job that failed must be attempted again, got %d attempts", got)
	}

	runs, err := s.Jobs().List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if runs[0].Status != domain.JobFailed || runs[0].Failures == 0 {
		t.Errorf("the failure must be recorded: %+v", runs[0])
	}
	if !strings.Contains(runs[0].Detail, "ERP refused") {
		t.Errorf("what went wrong is the whole reason to look: %q", runs[0].Detail)
	}
}

func TestAJobThatCouldNeverRunIsRefusedAtStartUp(t *testing.T) {
	s := memory.New()
	run := func(context.Context) (string, error) { return "", nil }

	cases := map[string]jobs.Job{
		"no interval": {Name: "dispatch", Run: run},
		"no name":     {Every: time.Minute, Run: run},
		"nothing to run": {
			Name: "dispatch", Every: time.Minute,
		},
	}
	for name, job := range cases {
		if _, err := jobs.New(s, "instance-a", quiet(), nil, job); err == nil {
			t.Errorf("%s: a job that could never run must be refused, not started", name)
		}
	}

	// Two jobs sharing a name would fight over one lease and each would see the
	// other's runs as its own.
	if _, err := jobs.New(s, "instance-a", quiet(), nil,
		jobs.Job{Name: "dispatch", Every: time.Minute, Run: run},
		jobs.Job{Name: "dispatch", Every: time.Minute, Run: run},
	); err == nil {
		t.Error("two jobs with the same name must be refused")
	}
}

func TestStopWaitsForTheRunInFlight(t *testing.T) {
	ctx := context.Background()
	s := memory.New()

	finished := make(chan struct{})
	scheduler, err := jobs.New(s, "instance-a", quiet(), nil, jobs.Job{
		Name: "slow", Every: 5 * time.Millisecond, Timeout: time.Second,
		Run: func(context.Context) (string, error) {
			time.Sleep(30 * time.Millisecond)
			close(finished)
			return "done", nil
		},
	})
	if err != nil {
		t.Fatalf("build scheduler: %v", err)
	}

	scheduler.Start(ctx)
	time.Sleep(10 * time.Millisecond)
	scheduler.Stop()

	select {
	case <-finished:
	default:
		t.Fatal("Stop must wait for the run in flight rather than abandoning it")
	}
	runs, err := s.Jobs().List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// The outcome is recorded even though the scheduler was shutting down: a run
	// that happened and was not recorded looks exactly like one that never ran.
	if len(runs) != 1 || runs[0].Runs != 1 || runs[0].Status != domain.JobOK {
		t.Errorf("the interrupted-looking run must still be recorded: %+v", runs)
	}
}
