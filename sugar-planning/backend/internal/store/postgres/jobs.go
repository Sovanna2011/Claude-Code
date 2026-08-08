package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type jobs struct{ s *Store }

// Jobs returns the scheduler's lease table.
func (s *Store) Jobs() store.Jobs { return jobs{s} }

// Acquire takes the lease in one statement, so two instances racing for the
// same job cannot both win.
//
// The upsert is the whole mechanism: the row is inserted if the job has never
// run, and updated only when nobody holds the lease or the holder's lease has
// expired. The number of rows affected is the answer - one means this instance
// has it, none means somebody else does.
func (j jobs) Acquire(ctx context.Context, name, owner string, lease time.Duration,
	now time.Time,
) (bool, error) {

	if lease <= 0 {
		lease = time.Minute
	}
	expires := now.Add(lease)

	tag, err := j.s.q.Exec(ctx, `INSERT INTO job_leases
		(name, owner, expires_at, started_at, status)
		VALUES ($1, $2, $3, $4, 'RUNNING')
		ON CONFLICT (name) DO UPDATE
		SET owner = EXCLUDED.owner, expires_at = EXCLUDED.expires_at,
		    started_at = EXCLUDED.started_at, finished_at = NULL, status = 'RUNNING'
		WHERE job_leases.status <> 'RUNNING'
		   OR job_leases.expires_at IS NULL
		   OR job_leases.expires_at <= $4
		   OR job_leases.owner = $2`,
		name, owner, expires, now)
	if err != nil {
		return false, mapError("job lease", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (j jobs) Finish(ctx context.Context, name, owner string, now time.Time,
	status, detail string,
) error {

	if len(detail) > 2000 {
		detail = detail[:2000]
	}
	failure := 0
	if status == domain.JobFailed {
		failure = 1
	}
	tag, err := j.s.q.Exec(ctx, `UPDATE job_leases
		SET finished_at = $1, status = $2, detail = $3, expires_at = NULL,
		    runs = runs + 1, failures = failures + $4
		WHERE name = $5 AND owner = $6`,
		now, status, detail, failure, name, owner)
	if err != nil {
		return mapError("job lease", err)
	}
	if tag.RowsAffected() == 0 {
		// Either the job is unknown or somebody else took the lease over after
		// it expired. Overwriting their record with this run's outcome would
		// misreport both.
		return fmt.Errorf("%w: job %s is not held by %s", domain.ErrConflict, name, owner)
	}
	return nil
}

func (j jobs) List(ctx context.Context) ([]domain.JobRun, error) {
	rows, err := j.s.q.Query(ctx, `SELECT name, owner, started_at, finished_at,
		status, detail, runs, failures FROM job_leases ORDER BY name`)
	if err != nil {
		return nil, mapError("job lease", err)
	}
	defer rows.Close()

	out := []domain.JobRun{}
	for rows.Next() {
		var run domain.JobRun
		if err := rows.Scan(&run.Name, &run.Owner, &run.StartedAt, &run.FinishedAt,
			&run.Status, &run.Detail, &run.Runs, &run.Failures); err != nil {
			return nil, mapError("job lease", err)
		}
		out = append(out, run)
	}
	return out, mapError("job lease", rows.Err())
}
