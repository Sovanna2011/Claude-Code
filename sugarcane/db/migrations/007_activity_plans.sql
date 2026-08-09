-- The activity plan: an approved projection laid out over the calendar, one row per activity per
-- block, with the dates the dependency chain and the working week allow.
--
-- The dates are generated, never typed. What is stored here is the output of one deterministic
-- function of three inputs — the projection's lines, the activity master, and the working calendar
-- on the header — so a plan can always be regenerated and will come out the same.

CREATE TYPE activity_plan_status AS ENUM ('Draft', 'Released', 'Closed');
CREATE TYPE plan_task_status     AS ENUM ('Planned', 'InProgress', 'Completed', 'Cancelled');

CREATE TABLE activity_plan (
    id            serial PRIMARY KEY,
    projection_id integer NOT NULL REFERENCES planting_projection(id) ON DELETE CASCADE,
    plan_no       text NOT NULL,

    -- The calendar the programme was laid out against, kept with the plan rather than read from a
    -- global. Regenerating a year-old plan reproduces it; a what-if scenario can vary it without
    -- touching what was agreed.
    work_on_saturday boolean NOT NULL DEFAULT true,
    work_on_sunday   boolean NOT NULL DEFAULT false,
    holidays         date[]  NOT NULL DEFAULT '{}',

    status       activity_plan_status NOT NULL DEFAULT 'Draft',
    starts_on    date,
    ends_on      date,

    -- Derived from the tasks by the same trigger that writes them; the list screen reads these
    -- rather than aggregating every task.
    task_count          integer       NOT NULL DEFAULT 0,
    total_area_ha       numeric(14,4) NOT NULL DEFAULT 0,
    total_working_hours numeric(14,2) NOT NULL DEFAULT 0,
    total_labour_days   numeric(14,2) NOT NULL DEFAULT 0,

    generated_at timestamptz,
    generated_by text,
    released_at  timestamptz,
    released_by  text,
    closed_at    timestamptz,
    closed_by    text,

    remark      text,
    row_version integer NOT NULL DEFAULT 1,

    created_at timestamptz NOT NULL DEFAULT now(),
    created_by text        NOT NULL DEFAULT 'system',
    updated_at timestamptz NOT NULL DEFAULT now(),
    updated_by text        NOT NULL DEFAULT 'system',

    -- One plan per projection. A projection that needs a different programme gets a revision, which
    -- is a different projection, and therefore a plan of its own.
    UNIQUE (projection_id),
    UNIQUE (plan_no),
    CONSTRAINT plan_window CHECK (ends_on IS NULL OR starts_on IS NULL OR ends_on >= starts_on)
);

CREATE INDEX activity_plan_status_idx ON activity_plan(status);

CREATE TABLE activity_plan_task (
    id                 bigserial PRIMARY KEY,
    plan_id            integer NOT NULL REFERENCES activity_plan(id) ON DELETE CASCADE,
    projection_line_id integer NOT NULL REFERENCES projection_line(id) ON DELETE CASCADE,
    block_id           integer NOT NULL REFERENCES block(id),
    activity_id        integer NOT NULL REFERENCES planting_activity(id),
    sequence_no        integer NOT NULL,

    planned_area_ha     numeric(12,4) NOT NULL,
    planned_start_date  date NOT NULL,
    planned_end_date    date NOT NULL,
    duration_days       integer NOT NULL,
    daily_target_ha     numeric(12,4) NOT NULL,
    planned_working_hours numeric(12,2) NOT NULL DEFAULT 0,
    required_labour_days  numeric(12,2) NOT NULL DEFAULT 0,
    required_workers      integer NOT NULL DEFAULT 0,

    -- Filled in by the execution module later; the plan itself only ever writes the planned side.
    status             plan_task_status NOT NULL DEFAULT 'Planned',
    completion_percent numeric(5,2) NOT NULL DEFAULT 0,
    actual_start_date  date,
    actual_end_date    date,
    actual_area_ha     numeric(12,4) NOT NULL DEFAULT 0,

    remark text,

    created_at timestamptz NOT NULL DEFAULT now(),
    created_by text        NOT NULL DEFAULT 'system',
    updated_at timestamptz NOT NULL DEFAULT now(),
    updated_by text        NOT NULL DEFAULT 'system',

    UNIQUE (plan_id, projection_line_id, activity_id),
    CONSTRAINT task_dates_ordered    CHECK (planned_end_date >= planned_start_date),
    CONSTRAINT task_duration_positive CHECK (duration_days >= 1),
    CONSTRAINT task_area_positive     CHECK (planned_area_ha > 0),
    CONSTRAINT task_completion_range  CHECK (completion_percent BETWEEN 0 AND 100),
    CONSTRAINT task_actual_dates_ordered
        CHECK (actual_end_date IS NULL OR actual_start_date IS NULL OR actual_end_date >= actual_start_date)
);

CREATE INDEX plan_task_plan_idx     ON activity_plan_task(plan_id, sequence_no);
CREATE INDEX plan_task_block_idx    ON activity_plan_task(block_id);
CREATE INDEX plan_task_activity_idx ON activity_plan_task(activity_id);
CREATE INDEX plan_task_window_idx   ON activity_plan_task(planned_start_date, planned_end_date);

-- ---------------------------------------------------------------- derived header figures
--
-- The same arrangement as the projection: the header is summed from its children by the database,
-- so it cannot drift from them whatever writes the tasks.

CREATE OR REPLACE FUNCTION recalculate_activity_plan(target integer) RETURNS void AS $$
BEGIN
    UPDATE activity_plan p
       SET starts_on           = agg.starts_on,
           ends_on             = agg.ends_on,
           task_count          = COALESCE(agg.tasks, 0),
           total_area_ha       = COALESCE(agg.area, 0),
           total_working_hours = COALESCE(agg.hours, 0),
           total_labour_days   = COALESCE(agg.labour, 0)
      FROM (SELECT min(planned_start_date) starts_on, max(planned_end_date) ends_on,
                   count(*) tasks, sum(planned_area_ha) area,
                   sum(planned_working_hours) hours, sum(required_labour_days) labour
              FROM activity_plan_task WHERE plan_id = target) agg
     WHERE p.id = target;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION activity_plan_task_changed() RETURNS trigger AS $$
BEGIN
    PERFORM recalculate_activity_plan(COALESCE(NEW.plan_id, OLD.plan_id));
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Not deferred, and it touches only the header, so it cannot fire itself.
CREATE TRIGGER activity_plan_totals_follow_the_tasks
    AFTER INSERT OR UPDATE OR DELETE ON activity_plan_task
    FOR EACH ROW EXECUTE FUNCTION activity_plan_task_changed();

-- ---------------------------------------------------------------- the audit stamp
--
-- Migration 006 attached the stamp to every table that existed then. These two are new, so they
-- get it here — the test that walks pg_catalog would fail otherwise, which is the point of it.

CREATE TRIGGER stamp_activity_plan      BEFORE INSERT OR UPDATE ON activity_plan
    FOR EACH ROW EXECUTE FUNCTION stamp_row_change();
CREATE TRIGGER stamp_activity_plan_task BEFORE INSERT OR UPDATE ON activity_plan_task
    FOR EACH ROW EXECUTE FUNCTION stamp_row_change();

INSERT INTO schema_migrations(version) VALUES ('007_activity_plans');
