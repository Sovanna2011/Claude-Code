-- Farm Area Monitoring — schema
--
-- The hierarchy is company → plantation → farm → zone → block, and the block is the only level
-- that owns an area figure. Farm and zone rows deliberately have no area columns at all: the
-- specification requires their totals to be derived from the blocks beneath them, and the surest
-- way to stop anyone entering one by hand is to give them nowhere to enter it.
--
-- Geometry is geography(MultiPolygon, 4326) — geography rather than geometry so ST_Area returns
-- square metres on the spheroid without anyone having to pick a projection for southern Zambia.

CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE schema_migrations (
    version     text PRIMARY KEY,
    applied_at  timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------- reference data

CREATE TABLE company (
    id          serial PRIMARY KEY,
    code        text NOT NULL UNIQUE,
    name        text NOT NULL,
    active      boolean NOT NULL DEFAULT true
);

CREATE TABLE plantation (
    id          serial PRIMARY KEY,
    company_id  integer NOT NULL REFERENCES company(id),
    code        text NOT NULL,
    name        text NOT NULL,
    active      boolean NOT NULL DEFAULT true,
    UNIQUE (company_id, code)
);

CREATE TABLE crop_season (
    id          serial PRIMARY KEY,
    code        text NOT NULL UNIQUE,
    name        text NOT NULL,
    crop_year   integer NOT NULL,
    starts_on   date NOT NULL,
    ends_on     date NOT NULL,
    active      boolean NOT NULL DEFAULT true,
    CONSTRAINT crop_season_dates CHECK (ends_on > starts_on)
);

CREATE TABLE cane_variety (
    id                    serial PRIMARY KEY,
    code                  text NOT NULL UNIQUE,
    name                  text NOT NULL,
    growing_period_months integer NOT NULL DEFAULT 12,
    active                boolean NOT NULL DEFAULT true,
    CONSTRAINT cane_variety_period CHECK (growing_period_months BETWEEN 1 AND 36)
);

-- The reasons a piece of land cannot carry cane (section 5).
CREATE TABLE non_plantable_reason (
    code        text PRIMARY KEY,
    name        text NOT NULL,
    sort_order  integer NOT NULL DEFAULT 0,
    active      boolean NOT NULL DEFAULT true
);

-- ---------------------------------------------------------------- hierarchy

CREATE TABLE farm (
    id            serial PRIMARY KEY,
    plantation_id integer NOT NULL REFERENCES plantation(id),
    code          text NOT NULL,
    name          text NOT NULL,
    manager_name  text,
    boundary      geography(MultiPolygon, 4326),
    remark        text,
    active        boolean NOT NULL DEFAULT true,
    version       integer NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL DEFAULT 'system',
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL DEFAULT 'system',
    UNIQUE (plantation_id, code)
);

CREATE TABLE zone (
    id               serial PRIMARY KEY,
    farm_id          integer NOT NULL REFERENCES farm(id),
    code             text NOT NULL,
    name             text NOT NULL,
    supervisor_name  text,
    boundary         geography(MultiPolygon, 4326),
    remark           text,
    active           boolean NOT NULL DEFAULT true,
    version          integer NOT NULL DEFAULT 1,
    created_at       timestamptz NOT NULL DEFAULT now(),
    created_by       text NOT NULL DEFAULT 'system',
    updated_at       timestamptz NOT NULL DEFAULT now(),
    updated_by       text NOT NULL DEFAULT 'system',
    UNIQUE (farm_id, code)
);

-- Land status is what the estate office says about the parcel; cane status is what is standing on
-- it today. Section 8 filters on both, and they answer different questions — a Reserved block can
-- still be carrying a crop that has to be harvested before the reservation takes effect.
CREATE TYPE land_status  AS ENUM ('Active', 'Reserved', 'Retired');
CREATE TYPE cane_status  AS ENUM ('Fallow', 'Prepared', 'Planted', 'Growing', 'ReadyForHarvest', 'Harvested');
CREATE TYPE planting_type AS ENUM ('NewPlanting', 'Ratoon');

CREATE TABLE block (
    id                   serial PRIMARY KEY,
    zone_id              integer NOT NULL REFERENCES zone(id),
    code                 text NOT NULL,
    name                 text NOT NULL,

    total_area_ha        numeric(12,4) NOT NULL,
    plantable_area_ha    numeric(12,4) NOT NULL,
    -- Never entered: the two columns above are the only truth about how the parcel divides.
    non_plantable_area_ha numeric(12,4) GENERATED ALWAYS AS (total_area_ha - plantable_area_ha) STORED,

    land_status          land_status NOT NULL DEFAULT 'Active',
    cane_status          cane_status NOT NULL DEFAULT 'Fallow',

    latitude             numeric(9,6),
    longitude            numeric(9,6),
    boundary             geography(MultiPolygon, 4326),

    remark               text,
    active               boolean NOT NULL DEFAULT true,
    version              integer NOT NULL DEFAULT 1,
    created_at           timestamptz NOT NULL DEFAULT now(),
    created_by           text NOT NULL DEFAULT 'system',
    updated_at           timestamptz NOT NULL DEFAULT now(),
    updated_by           text NOT NULL DEFAULT 'system',

    UNIQUE (zone_id, code),
    CONSTRAINT block_total_area_not_negative     CHECK (total_area_ha >= 0),
    CONSTRAINT block_plantable_area_not_negative CHECK (plantable_area_ha >= 0),
    CONSTRAINT block_plantable_within_total      CHECK (plantable_area_ha <= total_area_ha),
    CONSTRAINT block_latitude_range              CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
    CONSTRAINT block_longitude_range             CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180)
);

CREATE INDEX block_zone_idx ON block(zone_id);
CREATE INDEX block_boundary_idx ON block USING gist (boundary);
CREATE INDEX zone_farm_idx ON zone(farm_id);
CREATE INDEX farm_plantation_idx ON farm(plantation_id);

-- How the unplantable part of a block breaks down. The rows may cover part of it while a survey
-- is still in progress, so the rule is "no more than", not "exactly".
CREATE TABLE block_non_plantable (
    id           serial PRIMARY KEY,
    block_id     integer NOT NULL REFERENCES block(id) ON DELETE CASCADE,
    reason_code  text NOT NULL REFERENCES non_plantable_reason(code),
    area_ha      numeric(12,4) NOT NULL,
    remark       text,
    UNIQUE (block_id, reason_code),
    CONSTRAINT block_non_plantable_area_positive CHECK (area_ha > 0)
);

-- ---------------------------------------------------------------- planting

-- One row per block per crop year per planting type: what was planned, and what actually
-- happened. Both the dashboard's cane figures and the plan-versus-actual comparison read this
-- table, which is what keeps the two consistent.
CREATE TABLE block_planting (
    id                   serial PRIMARY KEY,
    block_id             integer NOT NULL REFERENCES block(id) ON DELETE CASCADE,
    crop_season_id       integer NOT NULL REFERENCES crop_season(id),
    crop_year            integer NOT NULL,
    planting_year        integer NOT NULL,
    planting_type        planting_type NOT NULL,
    ratoon_no            integer NOT NULL DEFAULT 0,
    cane_variety_id      integer REFERENCES cane_variety(id),

    planned_area_ha      numeric(12,4) NOT NULL DEFAULT 0,
    actual_area_ha       numeric(12,4) NOT NULL DEFAULT 0,
    planned_date         date,
    actual_date          date,
    expected_harvest_date date,

    remark               text,
    version              integer NOT NULL DEFAULT 1,
    created_at           timestamptz NOT NULL DEFAULT now(),
    created_by           text NOT NULL DEFAULT 'system',
    updated_at           timestamptz NOT NULL DEFAULT now(),
    updated_by           text NOT NULL DEFAULT 'system',

    UNIQUE (block_id, crop_year, planting_type),
    CONSTRAINT planting_planned_not_negative CHECK (planned_area_ha >= 0),
    CONSTRAINT planting_actual_not_negative  CHECK (actual_area_ha >= 0),
    CONSTRAINT planting_ratoon_no            CHECK (ratoon_no >= 0),
    CONSTRAINT planting_ratoon_number_matches_type
        CHECK ((planting_type = 'NewPlanting' AND ratoon_no = 0) OR (planting_type = 'Ratoon' AND ratoon_no >= 1))
);

CREATE INDEX block_planting_block_idx  ON block_planting(block_id);
CREATE INDEX block_planting_year_idx   ON block_planting(crop_year, planting_year);
CREATE INDEX block_planting_season_idx ON block_planting(crop_season_id);

-- Section 10 in one place. A block's cane can never exceed the land that can carry it, and the
-- rule spans rows — one new-planting row plus one ratoon row on the same block — so no single
-- CHECK can express it. A constraint trigger deferred to the end of the transaction lets a
-- caller move area between the two rows in either order without tripping over itself halfway.
-- Section 10 in one place. A block's cane can never exceed the land that can carry it, and the
-- rule spans rows — one new-planting row plus one ratoon row on the same block — so no single
-- CHECK can express it. A constraint trigger deferred to the end of the transaction lets a
-- caller move area between the two rows in either order without tripping over itself halfway.
CREATE OR REPLACE FUNCTION check_block_area_consistency(target_block integer) RETURNS void AS $$
DECLARE
    plantable      numeric(12,4);
    capacity       numeric(12,4);
    block_code     text;
    planned_total  numeric(12,4);
    actual_total   numeric(12,4);
    reasons_total  numeric(12,4);
BEGIN
    SELECT b.plantable_area_ha, b.non_plantable_area_ha, b.code
      INTO plantable, capacity, block_code
      FROM block b WHERE b.id = target_block;
    IF NOT FOUND THEN
        RETURN;                                   -- the block itself was deleted; nothing to check
    END IF;

    SELECT COALESCE(SUM(planned_area_ha), 0), COALESCE(SUM(actual_area_ha), 0)
      INTO planned_total, actual_total
      FROM block_planting WHERE block_id = target_block;

    IF planned_total > plantable THEN
        RAISE EXCEPTION 'AREA_EXCEEDS_PLANTABLE: planned area % ha on block % exceeds its plantable area of % ha',
            planned_total, block_code, plantable USING ERRCODE = 'check_violation';
    END IF;
    IF actual_total > plantable THEN
        RAISE EXCEPTION 'AREA_EXCEEDS_PLANTABLE: cane area % ha on block % exceeds its plantable area of % ha',
            actual_total, block_code, plantable USING ERRCODE = 'check_violation';
    END IF;

    SELECT COALESCE(SUM(area_ha), 0) INTO reasons_total
      FROM block_non_plantable WHERE block_id = target_block;

    IF reasons_total > capacity THEN
        RAISE EXCEPTION 'AREA_EXCEEDS_NON_PLANTABLE: reasons account for % ha on block % but only % ha is unplantable',
            reasons_total, block_code, capacity USING ERRCODE = 'check_violation';
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Three tables can break the same invariant, and each names the block differently.
CREATE OR REPLACE FUNCTION assert_child_area_consistency() RETURNS trigger AS $$
BEGIN
    PERFORM check_block_area_consistency(COALESCE(NEW.block_id, OLD.block_id));
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION assert_block_area_consistency() RETURNS trigger AS $$
BEGIN
    PERFORM check_block_area_consistency(NEW.id);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER block_planting_within_plantable
    AFTER INSERT OR UPDATE OR DELETE ON block_planting
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION assert_child_area_consistency();

CREATE CONSTRAINT TRIGGER block_non_plantable_within_block
    AFTER INSERT OR UPDATE OR DELETE ON block_non_plantable
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION assert_child_area_consistency();

-- Shrinking a block's plantable area must not leave more cane on it than it can hold, and
-- growing the plantable part must not leave the recorded reasons covering land that is now usable.
CREATE CONSTRAINT TRIGGER block_area_stays_consistent
    AFTER UPDATE OF total_area_ha, plantable_area_ha ON block
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION assert_block_area_consistency();

-- ---------------------------------------------------------------- identity and audit

CREATE TABLE app_user (
    id             serial PRIMARY KEY,
    username       text NOT NULL UNIQUE,
    password_hash  text NOT NULL,
    full_name      text NOT NULL,
    role           text NOT NULL,
    active         boolean NOT NULL DEFAULT true,
    last_login_at  timestamptz,
    CONSTRAINT app_user_role CHECK (role IN ('Admin', 'Manager', 'Planner', 'Viewer'))
);

CREATE TABLE audit_log (
    id          bigserial PRIMARY KEY,
    at          timestamptz NOT NULL DEFAULT now(),
    actor       text NOT NULL,
    action      text NOT NULL,
    entity      text NOT NULL,
    entity_id   text,
    before_data jsonb,
    after_data  jsonb,
    remote_addr text
);

CREATE INDEX audit_log_at_idx     ON audit_log(at DESC);
CREATE INDEX audit_log_entity_idx ON audit_log(entity, entity_id);

-- ---------------------------------------------------------------- derived area figures
--
-- block_area() is the single definition of the six area figures the dashboard reports. Every
-- consumer — KPI cards, charts, tree report, map, plan-versus-actual — selects from it, so the
-- numbers cannot drift apart between screens.
--
-- It is a function rather than a view because the planting filters (crop year, season, variety,
-- planting type) have to be applied *inside* the per-block aggregate. Applying them outside would
-- either drop blocks that have no matching planting — they still have land to report — or, if
-- joined row by row, count a block's total area once per planting row it happens to have.
--
-- The area with cane is the recorded planting on the block, new plus ratoon. "Available" is the
-- plantable land that has none. Both are clamped at zero rather than allowed to go negative,
-- though the constraint triggers above should already make that impossible.

CREATE FUNCTION block_area(
    p_crop_year     integer DEFAULT NULL,
    p_planting_year integer DEFAULT NULL,
    p_season_id     integer DEFAULT NULL,
    p_variety_id    integer DEFAULT NULL,
    p_planting_type text    DEFAULT NULL
)
RETURNS TABLE (
    block_id                     integer,
    block_code                   text,
    block_name                   text,
    zone_id                      integer,
    zone_code                    text,
    zone_name                    text,
    farm_id                      integer,
    farm_code                    text,
    farm_name                    text,
    plantation_id                integer,
    plantation_name              text,
    company_id                   integer,
    company_name                 text,
    total_area_ha                numeric,
    plantable_area_ha            numeric,
    non_plantable_area_ha        numeric,
    land_status                  land_status,
    cane_status                  cane_status,
    latitude                     numeric,
    longitude                    numeric,
    active                       boolean,
    new_planting_area_ha         numeric,
    ratoon_area_ha               numeric,
    area_with_cane_ha            numeric,
    available_area_ha            numeric,
    planned_new_planting_area_ha numeric,
    planned_ratoon_area_ha       numeric,
    planned_area_with_cane_ha    numeric,
    planned_available_area_ha    numeric
)
LANGUAGE sql STABLE AS $$
SELECT
    b.id                                   AS block_id,
    b.code                                 AS block_code,
    b.name                                 AS block_name,
    z.id                                   AS zone_id,
    z.code                                 AS zone_code,
    z.name                                 AS zone_name,
    f.id                                   AS farm_id,
    f.code                                 AS farm_code,
    f.name                                 AS farm_name,
    p.id                                   AS plantation_id,
    p.name                                 AS plantation_name,
    c.id                                   AS company_id,
    c.name                                 AS company_name,
    b.total_area_ha,
    b.plantable_area_ha,
    b.non_plantable_area_ha,
    b.land_status,
    b.cane_status,
    b.latitude,
    b.longitude,
    b.active,
    COALESCE(pl.new_planting_ha, 0)        AS new_planting_area_ha,
    COALESCE(pl.ratoon_ha, 0)              AS ratoon_area_ha,
    COALESCE(pl.new_planting_ha, 0) + COALESCE(pl.ratoon_ha, 0) AS area_with_cane_ha,
    GREATEST(b.plantable_area_ha - COALESCE(pl.new_planting_ha, 0) - COALESCE(pl.ratoon_ha, 0), 0)
                                           AS available_area_ha,
    COALESCE(pl.planned_new_planting_ha, 0) AS planned_new_planting_area_ha,
    COALESCE(pl.planned_ratoon_ha, 0)       AS planned_ratoon_area_ha,
    COALESCE(pl.planned_new_planting_ha, 0) + COALESCE(pl.planned_ratoon_ha, 0)
                                           AS planned_area_with_cane_ha,
    GREATEST(b.plantable_area_ha - COALESCE(pl.planned_new_planting_ha, 0) - COALESCE(pl.planned_ratoon_ha, 0), 0)
                                           AS planned_available_area_ha
FROM block b
JOIN zone z       ON z.id = b.zone_id
JOIN farm f       ON f.id = z.farm_id
JOIN plantation p ON p.id = f.plantation_id
JOIN company c    ON c.id = p.company_id
LEFT JOIN LATERAL (
    SELECT
        SUM(actual_area_ha)  FILTER (WHERE bp.planting_type = 'NewPlanting') AS new_planting_ha,
        SUM(actual_area_ha)  FILTER (WHERE bp.planting_type = 'Ratoon')      AS ratoon_ha,
        SUM(planned_area_ha) FILTER (WHERE bp.planting_type = 'NewPlanting') AS planned_new_planting_ha,
        SUM(planned_area_ha) FILTER (WHERE bp.planting_type = 'Ratoon')      AS planned_ratoon_ha
    FROM block_planting bp
    WHERE bp.block_id = b.id
      AND (p_crop_year     IS NULL OR bp.crop_year      = p_crop_year)
      AND (p_planting_year IS NULL OR bp.planting_year  = p_planting_year)
      AND (p_season_id     IS NULL OR bp.crop_season_id = p_season_id)
      AND (p_variety_id    IS NULL OR bp.cane_variety_id = p_variety_id)
      AND (p_planting_type IS NULL OR bp.planting_type  = p_planting_type::planting_type)
) pl ON true;
$$;

INSERT INTO schema_migrations(version) VALUES ('001_schema');
