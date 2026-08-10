-- Growing seasons and the planting activity master (specification sections 4 and 6).
--
-- A season already existed as a label on a planting record; planning needs it to be a real window
-- with a status, because "may this projection be planted in March?" is a question about the
-- season, not about the record.

ALTER TABLE crop_season
    ADD COLUMN planting_window_start date,
    ADD COLUMN planting_window_end   date,
    ADD COLUMN harvest_window_start  date,
    ADD COLUMN harvest_window_end    date,
    ADD COLUMN status text NOT NULL DEFAULT 'Open',
    ADD COLUMN remark text,
    ADD COLUMN version integer NOT NULL DEFAULT 1,
    ADD CONSTRAINT crop_season_status CHECK (status IN ('Planned', 'Open', 'Closed'));

-- The window may wrap the year end — October to February is an ordinary planting season south of
-- the equator — so the only rule the dates must obey is that each pair is ordered.
ALTER TABLE crop_season
    ADD CONSTRAINT crop_season_planting_window
        CHECK (planting_window_start IS NULL OR planting_window_end IS NULL
               OR planting_window_end >= planting_window_start),
    ADD CONSTRAINT crop_season_harvest_window
        CHECK (harvest_window_start IS NULL OR harvest_window_end IS NULL
               OR harvest_window_end >= harvest_window_start);

UPDATE crop_season SET
    planting_window_start = starts_on,
    planting_window_end   = starts_on + INTERVAL '180 days',
    harvest_window_start  = ends_on - INTERVAL '120 days',
    harvest_window_end    = ends_on;

ALTER TABLE cane_variety
    ADD COLUMN seed_rate_per_ha       numeric(12,4) NOT NULL DEFAULT 8,
    ADD COLUMN expected_yield_per_ha  numeric(12,4) NOT NULL DEFAULT 90,
    ADD COLUMN expected_loss_percent  numeric(5,2)  NOT NULL DEFAULT 5,
    ADD COLUMN remark text,
    ADD COLUMN version integer NOT NULL DEFAULT 1,
    ADD CONSTRAINT cane_variety_seed_rate CHECK (seed_rate_per_ha > 0),
    ADD CONSTRAINT cane_variety_yield     CHECK (expected_yield_per_ha >= 0),
    ADD CONSTRAINT cane_variety_loss      CHECK (expected_loss_percent BETWEEN 0 AND 100);

-- ---------------------------------------------------------------- activities

-- The categories of the specification's own activity list, in the order field work runs.
CREATE TYPE activity_category AS ENUM (
    'Survey', 'LandPreparation', 'SeedPreparation', 'Planting', 'Fertilization',
    'WeedControl', 'PestControl', 'Irrigation', 'CropCare', 'Harvesting', 'Other');

CREATE TYPE applicable_crop_type AS ENUM ('NewPlanting', 'Ratoon', 'Both');

-- The configurable activity master of section 6. Everything the engines need to turn an area into
-- a duration, a machine count, a material list and a workforce lives on this row.
CREATE TABLE planting_activity (
    id                       serial PRIMARY KEY,
    company_id               integer NOT NULL REFERENCES company(id),
    code                     text NOT NULL,
    name                     text NOT NULL,
    category                 activity_category NOT NULL,
    applicable_crop_type     applicable_crop_type NOT NULL DEFAULT 'Both',
    sequence_no              integer NOT NULL,

    -- Days from the projection's planting start. Negative is before it: land is cleared before
    -- the cane goes in, and the specification's own sample activities start at day -60.
    standard_start_day_offset integer NOT NULL DEFAULT 0,

    standard_capacity_per_hour numeric(12,4) NOT NULL DEFAULT 0,
    standard_capacity_per_day  numeric(12,4) NOT NULL DEFAULT 0,
    standard_duration_per_ha   numeric(12,4) NOT NULL DEFAULT 0,   -- hours per hectare
    standard_labour_days_per_ha numeric(12,4) NOT NULL DEFAULT 0,

    required_tractor_type      text,
    required_equipment_category text,

    is_mandatory       boolean NOT NULL DEFAULT true,
    requires_tractor   boolean NOT NULL DEFAULT false,
    requires_equipment boolean NOT NULL DEFAULT false,
    requires_material  boolean NOT NULL DEFAULT false,
    requires_labour    boolean NOT NULL DEFAULT false,
    allow_overlap      boolean NOT NULL DEFAULT false,

    remark     text,
    active     boolean NOT NULL DEFAULT true,
    version    integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_at timestamptz NOT NULL DEFAULT now(),
    updated_by text NOT NULL DEFAULT 'system',

    UNIQUE (company_id, code),
    CONSTRAINT activity_sequence_positive CHECK (sequence_no > 0),
    CONSTRAINT activity_capacity_not_negative
        CHECK (standard_capacity_per_hour >= 0 AND standard_capacity_per_day >= 0),
    CONSTRAINT activity_duration_not_negative
        CHECK (standard_duration_per_ha >= 0 AND standard_labour_days_per_ha >= 0),
    -- An activity the engine must schedule but cannot size is a plan with no duration.
    CONSTRAINT activity_needs_a_daily_capacity
        CHECK (NOT (requires_tractor OR requires_equipment) OR standard_capacity_per_day > 0)
);

CREATE INDEX planting_activity_sequence_idx ON planting_activity(company_id, sequence_no);

-- "Activity X waits for activity Y", with an optional lag. A blocking dependency stops the
-- successor from starting; an advisory one only warns, which is how an estate models a preference
-- it is willing to break under pressure.
CREATE TABLE activity_dependency (
    id             serial PRIMARY KEY,
    activity_id    integer NOT NULL REFERENCES planting_activity(id) ON DELETE CASCADE,
    depends_on_id  integer NOT NULL REFERENCES planting_activity(id) ON DELETE CASCADE,
    lag_days       integer NOT NULL DEFAULT 0,
    is_blocking    boolean NOT NULL DEFAULT true,
    remark         text,

    UNIQUE (activity_id, depends_on_id),
    CONSTRAINT dependency_not_self CHECK (activity_id <> depends_on_id),
    CONSTRAINT dependency_lag_not_negative CHECK (lag_days >= 0)
);

CREATE INDEX activity_dependency_activity_idx ON activity_dependency(activity_id);
CREATE INDEX activity_dependency_depends_idx  ON activity_dependency(depends_on_id);

-- A dependency that closes a loop makes the schedule unsatisfiable: every activity in the cycle
-- waits for another that waits for it. Postgres cannot express that as a CHECK, so a recursive
-- walk at commit time refuses it. Doing it here rather than only in the service means a data fix
-- applied with psql cannot create one either.
CREATE OR REPLACE FUNCTION assert_no_dependency_cycle() RETURNS trigger AS $$
DECLARE
    cycle_found boolean;
BEGIN
    WITH RECURSIVE chain(activity_id, depth) AS (
        SELECT NEW.depends_on_id, 1
         UNION ALL
        SELECT d.depends_on_id, chain.depth + 1
          FROM activity_dependency d
          JOIN chain ON chain.activity_id = d.activity_id
         WHERE chain.depth < 200
    )
    SELECT EXISTS (SELECT 1 FROM chain WHERE activity_id = NEW.activity_id) INTO cycle_found;

    IF cycle_found THEN
        RAISE EXCEPTION 'DEPENDENCY_CYCLE: activity % already comes after activity %, so this dependency would close a loop',
            NEW.depends_on_id, NEW.activity_id USING ERRCODE = 'check_violation';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER activity_dependency_acyclic
    AFTER INSERT OR UPDATE ON activity_dependency
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION assert_no_dependency_cycle();

INSERT INTO schema_migrations(version) VALUES ('003_activities');
