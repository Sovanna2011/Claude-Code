// Package jobs runs the recurring work of the application inside the server
// process.
//
// There is no separate worker deployment. A sugar mill runs one or two
// application instances on its own hardware, and a second thing to install,
// monitor and restart would be a cost with no return. What the design does take
// seriously is the consequence: two instances both have a scheduler, both wake
// at the same moment, and only one of them may do each piece of work. That is
// what the lease in the store is for.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Job is one recurring piece of work.
type Job struct {
	// Name is the lease key, so it must be stable across releases.
	Name string
	// Every is how often the job is attempted. A tick that finds the lease held
	// by another instance does nothing and waits for the next one.
	Every time.Duration
	// Timeout bounds one run. A job that hangs must not hold its lease until
	// the process is restarted.
	Timeout time.Duration
	// Run does the work and returns a line for the operations screen: what it
	// did, in the words somebody would use to describe it.
	Run func(ctx context.Context) (string, error)
}

// Scheduler runs jobs on their intervals.
type Scheduler struct {
	store store.Store
	// owner identifies this instance in the lease. It goes in the log and on
	// the operations screen, so that "which instance ran it" is answerable.
	owner  string
	logger *slog.Logger
	now    func() time.Time
	jobs   []Job
	// observe is called with the outcome of every run, for metrics. It is a
	// function rather than a dependency on the API package, which would be the
	// wrong way round: a scheduler that imported the HTTP layer could not be
	// run without one.
	observe func(job, result string, took time.Duration)

	mu      sync.Mutex
	started bool
	stop    context.CancelFunc
	done    sync.WaitGroup
}

// New builds a scheduler. Jobs with a zero or negative interval are refused
// rather than silently never run.
func New(s store.Store, owner string, logger *slog.Logger, now func() time.Time, jobs ...Job) (*Scheduler, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if owner == "" {
		owner = "instance"
	}
	seen := map[string]bool{}
	for _, j := range jobs {
		switch {
		case j.Name == "":
			return nil, fmt.Errorf("a scheduled job needs a name")
		case seen[j.Name]:
			return nil, fmt.Errorf("two jobs are both called %q, so they would fight over one lease", j.Name)
		case j.Every <= 0:
			return nil, fmt.Errorf("job %q has no interval, so it would never run", j.Name)
		case j.Run == nil:
			return nil, fmt.Errorf("job %q has nothing to run", j.Name)
		}
		seen[j.Name] = true
	}
	return &Scheduler{store: s, owner: owner, logger: logger, now: now, jobs: jobs}, nil
}

// Observe registers a callback for the outcome of every run.
//
// It is how the metrics registry learns that the outbox dispatcher has started
// failing - which is the thing an operator pages on, and which is invisible in
// a request rate because nobody makes a request for it.
func (s *Scheduler) Observe(fn func(job, result string, took time.Duration)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observe = fn
}

// Start begins the tickers. It returns immediately; Stop waits for the jobs in
// flight to finish.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started || len(s.jobs) == 0 {
		return
	}
	s.started = true

	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	s.stop = cancel

	for _, job := range s.jobs {
		s.done.Add(1)
		go s.loop(runCtx, job)
	}
	s.logger.Info("background jobs started", "owner", s.owner, "jobs", len(s.jobs))
}

// Stop cancels the tickers and waits for the runs in flight.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.started = false
	stop := s.stop
	s.mu.Unlock()

	stop()
	s.done.Wait()
	s.logger.Info("background jobs stopped", "owner", s.owner)
}

func (s *Scheduler) loop(ctx context.Context, job Job) {
	defer s.done.Done()

	ticker := time.NewTicker(job.Every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// A tick that arrived while the previous run was in flight can be
			// waiting alongside a cancelled context, and select would choose
			// between them at random. Shutdown wins: starting a run the
			// scheduler is about to stop for would delay it for no purpose.
			if ctx.Err() != nil {
				return
			}
			s.attempt(ctx, job)
		}
	}
}

// attempt takes the lease, runs the job and records what happened. Everything
// it can go wrong at is logged rather than propagated: a scheduler that dies
// because one job failed once would take the others down with it.
func (s *Scheduler) attempt(ctx context.Context, job Job) {
	lease := job.Timeout
	if lease <= 0 {
		lease = 5 * time.Minute
	}
	// The lease outlives the run by a margin, so a job that finishes just as its
	// timeout expires still owns the lease it is about to release.
	acquired, err := s.store.Jobs().Acquire(ctx, job.Name, s.owner, lease+time.Minute, s.now())
	if err != nil {
		s.logger.Error("could not take the job lease", "job", job.Name, "error", err)
		return
	}
	if !acquired {
		// Another instance is running it. This is the normal case in a pair, not
		// a problem, so it is not logged at anything louder than debug.
		s.logger.Debug("job is held by another instance", "job", job.Name)
		return
	}

	runCtx, cancel := context.WithTimeout(ctx, lease)
	startedAt := s.now()
	detail, runErr := job.Run(runCtx)
	cancel()
	took := s.now().Sub(startedAt)

	status := domain.JobOK
	if runErr != nil {
		status, detail = domain.JobFailed, runErr.Error()
		s.logger.Error("scheduled job failed", "job", job.Name, "error", runErr)
	} else if detail != "" {
		s.logger.Info("scheduled job ran", "job", job.Name, "detail", detail)
	}

	// The outcome is recorded even when the process is shutting down: a run that
	// happened and was not recorded looks, on the operations screen, exactly
	// like a scheduler that has stopped.
	if s.observe != nil {
		s.observe(job.Name, string(status), took)
	}

	finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer finishCancel()
	if err := s.store.Jobs().Finish(finishCtx, job.Name, s.owner, s.now(), status, detail); err != nil {
		s.logger.Error("could not record the job outcome", "job", job.Name, "error", err)
	}
}
