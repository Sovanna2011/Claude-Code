package memory

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type jobs struct{ s *Store }

// Jobs returns the scheduler's lease table.
func (s *Store) Jobs() store.Jobs { return jobs{s} }

func (j jobs) Acquire(_ context.Context, name, owner string, lease time.Duration,
	now time.Time,
) (bool, error) {

	j.s.lock()
	defer j.s.unlock()

	held, ok := j.s.d.jobs[name]
	if ok && held.Status == domain.JobRunning && held.Owner != owner {
		// Somebody else is running it. Their lease is stored in the expiry the
		// in-memory store keeps alongside; an expired one is taken over.
		if expiry, running := j.s.d.jobLeases[name]; running && now.Before(expiry) {
			return false, nil
		}
	}
	started := now
	held.Name, held.Owner, held.Status, held.StartedAt = name, owner, domain.JobRunning, &started
	held.FinishedAt = nil
	j.s.d.jobs[name] = held
	j.s.d.jobLeases[name] = now.Add(lease)
	return true, nil
}

func (j jobs) Finish(_ context.Context, name, owner string, now time.Time,
	status, detail string,
) error {

	j.s.lock()
	defer j.s.unlock()

	held, ok := j.s.d.jobs[name]
	if !ok {
		return fmt.Errorf("%w: job %s", domain.ErrNotFound, name)
	}
	if held.Owner != owner {
		// Finishing somebody else's run would overwrite their record with an
		// outcome that is not theirs.
		return fmt.Errorf("%w: job %s is held by %s", domain.ErrConflict, name, held.Owner)
	}
	finished := now
	held.FinishedAt, held.Status, held.Detail = &finished, status, detail
	held.Runs++
	if status == domain.JobFailed {
		held.Failures++
	}
	j.s.d.jobs[name] = held
	delete(j.s.d.jobLeases, name)
	return nil
}

func (j jobs) List(_ context.Context) ([]domain.JobRun, error) {
	j.s.lock()
	defer j.s.unlock()

	out := make([]domain.JobRun, 0, len(j.s.d.jobs))
	for _, run := range j.s.d.jobs {
		out = append(out, run)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Name < out[b].Name })
	return out, nil
}
