-- 0002_planning: seasons, planning versions and the daily plan facts.
--
-- The daily tables are the heart of the system and replace the wide worksheet.
-- Each is narrow, keyed on (version, business date, dimensions, series) and
-- carries only the measures that belong to that part of the process.
--
-- `series` separates PLAN from ACTUAL rows. Actuals live in the season's
-- ACTUAL version, so posting an actual can never overwrite an approved plan
-- even if a defect let it try.

CREATE TABLE seasons (
    id           uuid PRIMARY KEY,
    company_id   uuid NOT NULL REFERENCES companies (id),
    factory_id   uuid NOT NULL REFERENCES factories (id),
    code         text NOT NULL,
    name         text NOT NULL,
    start_date   date NOT NULL,
    end_date     date,
    planned_days integer NOT NULL DEFAULT 0,
    status       text NOT NULL DEFAULT 'OPEN',
    created_at   timestamptz NOT NULL DEFAULT now(),
    created_by   text NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    updated_by   text NOT NULL,
    row_version  bigint NOT NULL DEFAULT 1,
    CONSTRAINT seasons_key UNIQUE (factory_id, code),
    CONSTRAINT seasons_days_ck CHECK (planned_days >= 0),
    CONSTRAINT seasons_status_ck CHECK (status IN ('OPEN','CLOSED')),
    CONSTRAINT seasons_dates_ck CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE TABLE plan_versions (
    id                uuid PRIMARY KEY,
    season_id         uuid NOT NULL REFERENCES seasons (id),
    version_no        integer NOT NULL,
    code              text NOT NULL,
    description       text NOT NULL DEFAULT '',
    plan_type         text NOT NULL,
    status            text NOT NULL DEFAULT 'DRAFT',
    owner             text NOT NULL DEFAULT '',
    source_version_id uuid REFERENCES plan_versions (id),
    effective_from    date,
    effective_to      date,
    submitted_at      timestamptz,
    approved_at       timestamptz,
    approved_by       text,
    released_at       timestamptz,
    locked_through    date,
    comment           text NOT NULL DEFAULT '',
    created_at        timestamptz NOT NULL DEFAULT now(),
    created_by        text NOT NULL,
    updated_at        timestamptz NOT NULL DEFAULT now(),
    updated_by        text NOT NULL,
    row_version       bigint NOT NULL DEFAULT 1,
    CONSTRAINT plan_versions_code_key UNIQUE (season_id, code),
    CONSTRAINT plan_versions_no_key UNIQUE (season_id, version_no),
    CONSTRAINT plan_versions_type_ck CHECK (plan_type IN
        ('BUDGET','FORECAST','REVISED','WHATIF','BASELINE','ACTUAL','LATEST_ESTIMATE')),
    CONSTRAINT plan_versions_status_ck CHECK (status IN
        ('DRAFT','IN_REVIEW','APPROVED','RELEASED','SUPERSEDED','REJECTED','CLOSED')),
    CONSTRAINT plan_versions_no_ck CHECK (version_no > 0)
);
CREATE INDEX plan_versions_season_status_idx ON plan_versions (season_id, status);
-- At most one released version per season: the released baseline is singular.
CREATE UNIQUE INDEX plan_versions_one_released ON plan_versions (season_id)
    WHERE status = 'RELEASED';
-- Exactly one actuals container per season.
CREATE UNIQUE INDEX plan_versions_one_actual ON plan_versions (season_id)
    WHERE plan_type = 'ACTUAL';

-- Every rate, factor and capacity the calculations use, effective dated so a
-- mid-season change to the recovery assumption does not rewrite history.
CREATE TABLE plan_assumptions (
    id          uuid PRIMARY KEY,
    version_id  uuid NOT NULL REFERENCES plan_versions (id) ON DELETE CASCADE,
    code        text NOT NULL,
    description text NOT NULL DEFAULT '',
    value       numeric(18,6) NOT NULL,
    uom         text NOT NULL DEFAULT '',
    valid_from  date,
    valid_to    date,
    created_at  timestamptz NOT NULL DEFAULT now(),
    created_by  text NOT NULL,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    updated_by  text NOT NULL,
    row_version bigint NOT NULL DEFAULT 1,
    CONSTRAINT plan_assumptions_dates_ck CHECK (valid_to IS NULL OR valid_from IS NULL OR valid_to >= valid_from)
);
CREATE UNIQUE INDEX plan_assumptions_key ON plan_assumptions
    (version_id, code, COALESCE(valid_from, DATE '0001-01-01'));

CREATE TABLE product_mix_entries (
    id              uuid PRIMARY KEY,
    version_id      uuid NOT NULL REFERENCES plan_versions (id) ON DELETE CASCADE,
    product_id      uuid NOT NULL REFERENCES products (id),
    packaging_id    uuid REFERENCES packaging_types (id),
    warehouse_id    uuid REFERENCES warehouses (id),
    line_id         uuid REFERENCES production_lines (id),
    season_tons     numeric(18,3) NOT NULL DEFAULT 0,
    daily_rate_tons numeric(18,3) NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    created_by      text NOT NULL,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    updated_by      text NOT NULL,
    row_version     bigint NOT NULL DEFAULT 1,
    CONSTRAINT product_mix_tons_ck CHECK (season_tons >= 0 AND daily_rate_tons >= 0)
);
CREATE UNIQUE INDEX product_mix_key ON product_mix_entries
    (version_id, product_id, COALESCE(packaging_id, '00000000-0000-0000-0000-000000000000'::uuid));

-- --------------------------------------------------------------------------
-- Daily facts
-- --------------------------------------------------------------------------

CREATE TABLE daily_cane_plans (
    id             uuid PRIMARY KEY,
    version_id     uuid NOT NULL REFERENCES plan_versions (id) ON DELETE CASCADE,
    factory_id     uuid NOT NULL REFERENCES factories (id),
    business_date  date NOT NULL,
    shift_id       uuid REFERENCES shifts (id),
    series         text NOT NULL,
    cane_available numeric(18,3) NOT NULL DEFAULT 0,
    cane_delivered numeric(18,3) NOT NULL DEFAULT 0,
    cane_accepted  numeric(18,3) NOT NULL DEFAULT 0,
    cane_rejected  numeric(18,3) NOT NULL DEFAULT 0,
    cane_diverted  numeric(18,3) NOT NULL DEFAULT 0,
    cane_crushed   numeric(18,3) NOT NULL DEFAULT 0,
    crush_rate_tph numeric(12,3) NOT NULL DEFAULT 0,
    available_hrs  numeric(6,2) NOT NULL DEFAULT 24,
    stoppage_hrs   numeric(6,2) NOT NULL DEFAULT 0,
    reason_code    text REFERENCES reason_codes (code),
    note           text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now(),
    created_by     text NOT NULL,
    updated_at     timestamptz NOT NULL DEFAULT now(),
    updated_by     text NOT NULL,
    row_version    bigint NOT NULL DEFAULT 1,
    CONSTRAINT daily_cane_series_ck CHECK (series IN ('PLAN','ACTUAL')),
    CONSTRAINT daily_cane_nonneg_ck CHECK (
        cane_available >= 0 AND cane_delivered >= 0 AND cane_accepted >= 0 AND
        cane_rejected >= 0 AND cane_diverted >= 0 AND cane_crushed >= 0 AND
        crush_rate_tph >= 0),
    CONSTRAINT daily_cane_hours_ck CHECK (
        available_hrs >= 0 AND available_hrs <= 24 AND stoppage_hrs >= 0 AND stoppage_hrs <= 24)
);
CREATE UNIQUE INDEX daily_cane_key ON daily_cane_plans
    (version_id, factory_id, business_date,
     COALESCE(shift_id, '00000000-0000-0000-0000-000000000000'::uuid), series);
CREATE INDEX daily_cane_date_idx ON daily_cane_plans (version_id, business_date);

CREATE TABLE daily_product_plans (
    id            uuid PRIMARY KEY,
    version_id    uuid NOT NULL REFERENCES plan_versions (id) ON DELETE CASCADE,
    factory_id    uuid NOT NULL REFERENCES factories (id),
    line_id       uuid REFERENCES production_lines (id),
    business_date date NOT NULL,
    shift_id      uuid REFERENCES shifts (id),
    product_id    uuid NOT NULL REFERENCES products (id),
    packaging_id  uuid REFERENCES packaging_types (id),
    series        text NOT NULL,
    quantity      numeric(18,3) NOT NULL DEFAULT 0,
    remelt_input  numeric(18,3) NOT NULL DEFAULT 0,
    process_loss  numeric(18,3) NOT NULL DEFAULT 0,
    rework        numeric(18,3) NOT NULL DEFAULT 0,
    rejected      numeric(18,3) NOT NULL DEFAULT 0,
    hold_qty      numeric(18,3) NOT NULL DEFAULT 0,
    reason_code   text REFERENCES reason_codes (code),
    note          text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT daily_product_series_ck CHECK (series IN ('PLAN','ACTUAL')),
    CONSTRAINT daily_product_nonneg_ck CHECK (
        quantity >= 0 AND remelt_input >= 0 AND process_loss >= 0 AND
        rework >= 0 AND rejected >= 0 AND hold_qty >= 0)
);
CREATE UNIQUE INDEX daily_product_key ON daily_product_plans
    (version_id, factory_id,
     COALESCE(line_id, '00000000-0000-0000-0000-000000000000'::uuid),
     business_date,
     COALESCE(shift_id, '00000000-0000-0000-0000-000000000000'::uuid),
     product_id,
     COALESCE(packaging_id, '00000000-0000-0000-0000-000000000000'::uuid),
     series);
CREATE INDEX daily_product_date_idx ON daily_product_plans (version_id, business_date, product_id);

CREATE TABLE daily_storage_plans (
    id                 uuid PRIMARY KEY,
    version_id         uuid NOT NULL REFERENCES plan_versions (id) ON DELETE CASCADE,
    warehouse_id       uuid NOT NULL REFERENCES warehouses (id),
    product_id         uuid NOT NULL REFERENCES products (id),
    business_date      date NOT NULL,
    series             text NOT NULL,
    beginning_balance  numeric(18,3) NOT NULL DEFAULT 0,
    production_receipt numeric(18,3) NOT NULL DEFAULT 0,
    transfer_in        numeric(18,3) NOT NULL DEFAULT 0,
    transfer_out       numeric(18,3) NOT NULL DEFAULT 0,
    repack_in          numeric(18,3) NOT NULL DEFAULT 0,
    repack_out         numeric(18,3) NOT NULL DEFAULT 0,
    remelt_issue       numeric(18,3) NOT NULL DEFAULT 0,
    shipment_qty       numeric(18,3) NOT NULL DEFAULT 0,
    adjustment         numeric(18,3) NOT NULL DEFAULT 0,
    process_loss       numeric(18,3) NOT NULL DEFAULT 0,
    hold_qty           numeric(18,3) NOT NULL DEFAULT 0,
    ending_balance     numeric(18,3) NOT NULL DEFAULT 0,
    physical_balance   numeric(18,3),
    created_at         timestamptz NOT NULL DEFAULT now(),
    created_by         text NOT NULL,
    updated_at         timestamptz NOT NULL DEFAULT now(),
    updated_by         text NOT NULL,
    row_version        bigint NOT NULL DEFAULT 1,
    CONSTRAINT daily_storage_series_ck CHECK (series IN ('PLAN','ACTUAL')),
    -- adjustment is signed; every other movement is a magnitude.
    CONSTRAINT daily_storage_nonneg_ck CHECK (
        production_receipt >= 0 AND transfer_in >= 0 AND transfer_out >= 0 AND
        repack_in >= 0 AND repack_out >= 0 AND remelt_issue >= 0 AND
        shipment_qty >= 0 AND process_loss >= 0 AND hold_qty >= 0),
    CONSTRAINT daily_storage_physical_ck CHECK (physical_balance IS NULL OR physical_balance >= 0)
);
CREATE UNIQUE INDEX daily_storage_key ON daily_storage_plans
    (version_id, warehouse_id, product_id, business_date, series);
CREATE INDEX daily_storage_date_idx ON daily_storage_plans (version_id, business_date, warehouse_id);

CREATE TABLE daily_shipment_plans (
    id            uuid PRIMARY KEY,
    version_id    uuid NOT NULL REFERENCES plan_versions (id) ON DELETE CASCADE,
    warehouse_id  uuid REFERENCES warehouses (id),
    product_id    uuid NOT NULL REFERENCES products (id),
    channel_id    uuid NOT NULL REFERENCES shipment_channels (id),
    business_date date NOT NULL,
    series        text NOT NULL,
    quantity      numeric(18,3) NOT NULL DEFAULT 0,
    note          text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT daily_shipment_series_ck CHECK (series IN ('PLAN','ACTUAL')),
    CONSTRAINT daily_shipment_qty_ck CHECK (quantity >= 0)
);
CREATE UNIQUE INDEX daily_shipment_key ON daily_shipment_plans
    (version_id, COALESCE(warehouse_id, '00000000-0000-0000-0000-000000000000'::uuid),
     product_id, channel_id, business_date, series);
CREATE INDEX daily_shipment_date_idx ON daily_shipment_plans (version_id, business_date, channel_id);

-- Recorded comparisons between two versions, so an analysis a planner ran can
-- be reopened later with the same parameters.
CREATE TABLE scenario_comparisons (
    id            uuid PRIMARY KEY,
    season_id     uuid NOT NULL REFERENCES seasons (id),
    base_version  uuid NOT NULL REFERENCES plan_versions (id),
    other_version uuid NOT NULL REFERENCES plan_versions (id),
    name          text NOT NULL,
    parameters    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT scenario_comparisons_distinct_ck CHECK (base_version <> other_version)
);
