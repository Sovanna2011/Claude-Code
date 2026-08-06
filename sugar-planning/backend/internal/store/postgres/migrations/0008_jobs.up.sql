-- 0008_jobs: the lease table the background scheduler runs behind.
--
-- Two instances of this application behind a load balancer both have a
-- scheduler, and both wake at the same moment. Without a lease they would both
-- evaluate the same alerts and write the same notifications twice. The lease is
-- deliberately a plain row rather than a PostgreSQL advisory lock: an advisory
-- lock dies with its connection, which is right for a lock and wrong for a
-- record of when a job last ran and what it said.
CREATE TABLE job_leases (
    name        text PRIMARY KEY,
    owner       text NOT NULL DEFAULT '',
    -- expires_at is what makes a crash recoverable: an instance that dies
    -- holding the lease blocks the job only until the lease runs out, rather
    -- than until somebody notices.
    expires_at  timestamptz,
    started_at  timestamptz,
    finished_at timestamptz,
    status      text NOT NULL DEFAULT '',
    detail      text NOT NULL DEFAULT '',
    runs        bigint NOT NULL DEFAULT 0,
    failures    bigint NOT NULL DEFAULT 0,
    CONSTRAINT job_leases_status_ck CHECK (status IN ('', 'RUNNING', 'OK', 'FAILED'))
);
