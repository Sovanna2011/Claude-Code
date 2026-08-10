-- Planting projections (specification sections 5, 18 and 20): the committed plan of what will be
-- planted, where, when and with which variety — and the approval workflow that commits it.

CREATE TYPE projection_status AS ENUM (
    'Draft', 'Submitted', 'UnderReview', 'Approved', 'Rejected', 'Revised', 'Closed');

CREATE TABLE planting_projection (
    id               serial PRIMARY KEY,
    company_id       integer NOT NULL REFERENCES company(id),
    plantation_id    integer NOT NULL REFERENCES plantation(id),
    crop_season_id   integer NOT NULL REFERENCES crop_season(id),

    projection_no    text NOT NULL,
    version          integer NOT NULL DEFAULT 1,
    -- The chain of revisions. A revision points at the projection it replaced, so the history is
    -- walkable in both directions without a separate table.
    supersedes_id    integer REFERENCES planting_projection(id),
    is_current       boolean NOT NULL DEFAULT true,

    projection_date  date NOT NULL,
    planning_start   date NOT NULL,
    planning_end     date NOT NULL,

    -- Derived from the lines by a trigger, never entered. Section 20 is explicit that a header
    -- total is calculated; storing it makes the list screen a single query instead of a join.
    total_projected_area_ha    numeric(14,4) NOT NULL DEFAULT 0,
    total_harvestable_area_ha  numeric(14,4) NOT NULL DEFAULT 0,
    total_expected_tons        numeric(16,4) NOT NULL DEFAULT 0,
    line_count                 integer NOT NULL DEFAULT 0,

    status           projection_status NOT NULL DEFAULT 'Draft',
    prepared_by      text,
    submitted_by     text,
    submitted_at     timestamptz,
    reviewed_by      text,
    reviewed_at      timestamptz,
    approved_by      text,
    approved_at      timestamptz,
    rejected_by      text,
    rejected_at      timestamptz,
    rejection_reason text,
    revision_reason  text,
    closed_by        text,
    closed_at        timestamptz,

    remark      text,
    row_version integer NOT NULL DEFAULT 1,
    created_at  timestamptz NOT NULL DEFAULT now(),
    created_by  text NOT NULL DEFAULT 'system',
    updated_at  timestamptz NOT NULL DEFAULT now(),
    updated_by  text NOT NULL DEFAULT 'system',

    UNIQUE (company_id, projection_no, version),
    CONSTRAINT projection_planning_window CHECK (planning_end >= planning_start),
    CONSTRAINT projection_version_positive CHECK (version > 0),
    CONSTRAINT projection_rejection_has_a_reason
        CHECK (status <> 'Rejected' OR rejection_reason IS NOT NULL)
);

CREATE INDEX projection_season_idx  ON planting_projection(crop_season_id);
CREATE INDEX projection_status_idx  ON planting_projection(status);
CREATE INDEX projection_current_idx ON planting_projection(company_id, projection_no) WHERE is_current;

CREATE TABLE projection_line (
    id             serial PRIMARY KEY,
    projection_id  integer NOT NULL REFERENCES planting_projection(id) ON DELETE CASCADE,
    block_id       integer NOT NULL REFERENCES block(id),
    cane_variety_id integer NOT NULL REFERENCES cane_variety(id),
    planting_type  planting_type NOT NULL DEFAULT 'NewPlanting',

    -- The block's plantable area as it stood when the line was written. Keeping it makes an
    -- approved plan readable years later even after the block has been re-surveyed.
    available_area_ha        numeric(12,4) NOT NULL,
    projected_planting_area_ha numeric(12,4) NOT NULL,

    planned_planting_start date NOT NULL,
    planned_planting_end   date NOT NULL,
    expected_harvest_date  date,

    expected_yield_per_ha  numeric(12,4) NOT NULL,
    expected_loss_percent  numeric(5,2)  NOT NULL DEFAULT 0,

    -- Derived by the same trigger that totals the header.
    harvestable_area_ha      numeric(12,4) NOT NULL DEFAULT 0,
    expected_production_tons numeric(16,4) NOT NULL DEFAULT 0,

    priority integer NOT NULL DEFAULT 5,
    remark   text,

    UNIQUE (projection_id, block_id, planting_type),
    CONSTRAINT line_area_positive     CHECK (projected_planting_area_ha > 0),
    CONSTRAINT line_area_within_block CHECK (projected_planting_area_ha <= available_area_ha),
    CONSTRAINT line_dates_ordered     CHECK (planned_planting_end >= planned_planting_start),
    CONSTRAINT line_loss_range        CHECK (expected_loss_percent BETWEEN 0 AND 100),
    CONSTRAINT line_yield_not_negative CHECK (expected_yield_per_ha >= 0),
    CONSTRAINT line_priority_range    CHECK (priority BETWEEN 1 AND 9)
);

CREATE INDEX projection_line_projection_idx ON projection_line(projection_id);
CREATE INDEX projection_line_block_idx      ON projection_line(block_id);

-- Every step of the workflow, with who took it and why. This is the trail an auditor reads.
CREATE TABLE projection_approval (
    id            bigserial PRIMARY KEY,
    projection_id integer NOT NULL REFERENCES planting_projection(id) ON DELETE CASCADE,
    action        text NOT NULL,
    from_status   projection_status NOT NULL,
    to_status     projection_status NOT NULL,
    actor         text NOT NULL,
    at            timestamptz NOT NULL DEFAULT now(),
    comments      text,
    CONSTRAINT approval_action CHECK (action IN
        ('Submit', 'Review', 'Approve', 'Reject', 'ReturnForCorrection', 'Revise', 'Close'))
);

CREATE INDEX projection_approval_projection_idx ON projection_approval(projection_id, at);

-- ---------------------------------------------------------------- derived figures
--
-- Section 20: header totals are always derived from the lines, never entered. A trigger rather
-- than application code, so a line inserted by any route — the API, a migration, psql — leaves the
-- header telling the truth.

-- Harvestable area = projected area x (1 - loss / 100); production = harvestable x yield. The same
-- formulas as domain.HarvestableArea and domain.ExpectedCaneProduction in Go, and the tests assert
-- the two agree.
--
-- BEFORE, so the row is derived on its way in. Deriving it afterwards would mean this trigger's
-- function updating the very table the trigger is on, and it would fire itself for ever.
CREATE OR REPLACE FUNCTION projection_line_derive() RETURNS trigger AS $$
BEGIN
    NEW.harvestable_area_ha :=
        round(NEW.projected_planting_area_ha * (1 - NEW.expected_loss_percent / 100), 4);
    NEW.expected_production_tons :=
        round(NEW.harvestable_area_ha * NEW.expected_yield_per_ha, 4);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER projection_line_derived_figures
    BEFORE INSERT OR UPDATE ON projection_line
    FOR EACH ROW EXECUTE FUNCTION projection_line_derive();

CREATE OR REPLACE FUNCTION recalculate_projection(target integer) RETURNS void AS $$
BEGIN
    UPDATE planting_projection p
       SET total_projected_area_ha   = COALESCE(agg.area, 0),
           total_harvestable_area_ha = COALESCE(agg.harvestable, 0),
           total_expected_tons       = COALESCE(agg.tons, 0),
           line_count                = COALESCE(agg.lines, 0)
      FROM (SELECT SUM(projected_planting_area_ha) area,
                   SUM(harvestable_area_ha) harvestable,
                   SUM(expected_production_tons) tons,
                   COUNT(*) lines
              FROM projection_line WHERE projection_id = target) agg
     WHERE p.id = target;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION projection_line_changed() RETURNS trigger AS $$
BEGIN
    PERFORM recalculate_projection(COALESCE(NEW.projection_id, OLD.projection_id));
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Not deferred, unlike the area rule below. A deferred trigger does not run until COMMIT, so the
-- header would read zero for the whole transaction that wrote the lines — and the caller that just
-- added a line wants the new totals back in the same transaction. Recalculation is idempotent and
-- order-independent, so there is nothing to gain by putting it off.
CREATE TRIGGER projection_totals_follow_the_lines
    AFTER INSERT OR UPDATE OR DELETE ON projection_line
    FOR EACH ROW EXECUTE FUNCTION projection_line_changed();

-- ---------------------------------------------------------------- the two area rules
--
-- Section 5: a line may not exceed its block's plantable area, and neither may the sum of the
-- lines for the same block. The first is a CHECK; the second spans rows and is not.

CREATE OR REPLACE FUNCTION assert_projection_area_within_block() RETURNS trigger AS $$
DECLARE
    target_line   record;
    plantable     numeric(12,4);
    block_code    text;
    committed     numeric(12,4);
BEGIN
    SELECT * INTO target_line FROM projection_line WHERE id = COALESCE(NEW.id, OLD.id);
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    SELECT b.plantable_area_ha, b.code INTO plantable, block_code
      FROM block b WHERE b.id = target_line.block_id;

    -- Every line on this block, in this projection, whatever the planting type.
    SELECT COALESCE(SUM(projected_planting_area_ha), 0) INTO committed
      FROM projection_line
     WHERE projection_id = target_line.projection_id AND block_id = target_line.block_id;

    IF committed > plantable THEN
        RAISE EXCEPTION 'AREA_EXCEEDS_BLOCK: the lines for block % total % ha but only % ha is plantable',
            block_code, committed, plantable USING ERRCODE = 'check_violation';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER projection_line_within_block
    AFTER INSERT OR UPDATE ON projection_line
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION assert_projection_area_within_block();

-- Section 5 again: committed plans for one block may not overlap in time. Two drafts on the same
-- block are allowed — planners work in parallel — and the rule bites on approval, when the block
-- is actually taken. Only Approved projections count.

CREATE OR REPLACE FUNCTION assert_no_committed_overlap(target integer) RETURNS void AS $$
DECLARE
    clash record;
BEGIN
    SELECT b.code AS block_code, other.projection_no, other.version,
           l.planned_planting_start, l.planned_planting_end
      INTO clash
      FROM projection_line mine
      JOIN block b ON b.id = mine.block_id
      JOIN projection_line l ON l.block_id = mine.block_id AND l.projection_id <> mine.projection_id
      JOIN planting_projection other ON other.id = l.projection_id
     WHERE mine.projection_id = target
       AND other.status = 'Approved'
       AND other.is_current
       AND mine.planned_planting_start <= l.planned_planting_end
       AND l.planned_planting_start <= mine.planned_planting_end
     LIMIT 1;

    IF FOUND THEN
        RAISE EXCEPTION 'BLOCK_ALREADY_COMMITTED: block % is already committed to % v% from % to %',
            clash.block_code, clash.projection_no, clash.version,
            clash.planned_planting_start, clash.planned_planting_end
            USING ERRCODE = 'check_violation';
    END IF;
END;
$$ LANGUAGE plpgsql;

INSERT INTO schema_migrations(version) VALUES ('005_projections');
