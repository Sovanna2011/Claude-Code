-- Where the cane comes from.
--
-- Section 5 asks the crushing plan to carry "cane source/zone/farm, delivery
-- schedule, transport capacity, queue, and expected quality". The plan could
-- say how much cane would be crushed each day and not where a ton of it came
-- from: the season target was an assumption with nothing behind it.

CREATE TABLE cane_sources (
    id                  uuid PRIMARY KEY,
    factory_id          uuid NOT NULL REFERENCES factories (id),
    code                text NOT NULL,
    name                text NOT NULL,
    source_type         text NOT NULL,
    zone                text NOT NULL DEFAULT '',
    distance_km         numeric(9,3) NOT NULL DEFAULT 0,
    hectares            numeric(12,3) NOT NULL DEFAULT 0,
    variety             text NOT NULL DEFAULT '',
    expected_yield_tph  numeric(9,3) NOT NULL DEFAULT 0,
    expected_pol_pct    numeric(6,3) NOT NULL DEFAULT 0,
    truck_capacity_tons numeric(9,3) NOT NULL DEFAULT 0,
    trucks_per_day      integer NOT NULL DEFAULT 0,
    valid_from          date,
    valid_to            date,
    active              boolean NOT NULL DEFAULT true,
    created_at          timestamptz NOT NULL DEFAULT now(),
    created_by          text NOT NULL,
    updated_at          timestamptz NOT NULL DEFAULT now(),
    updated_by          text NOT NULL,
    row_version         bigint NOT NULL DEFAULT 1,
    CONSTRAINT cane_sources_key UNIQUE (factory_id, code),
    CONSTRAINT cane_sources_type_ck CHECK (source_type IN ('ESTATE','CONTRACT','OUTGROWER')),
    CONSTRAINT cane_sources_nonneg_ck CHECK (
        distance_km >= 0 AND hectares >= 0 AND expected_yield_tph >= 0 AND
        expected_pol_pct >= 0 AND expected_pol_pct <= 100 AND
        truck_capacity_tons >= 0 AND trucks_per_day >= 0)
);
CREATE INDEX cane_sources_factory_idx ON cane_sources (factory_id, active);

-- What one source is committed to for one season, and the window it is cut in.
-- The cane-side counterpart of product_mix_entries.
CREATE TABLE cane_supply_entries (
    id             uuid PRIMARY KEY,
    version_id     uuid NOT NULL REFERENCES plan_versions (id) ON DELETE CASCADE,
    source_id      uuid NOT NULL REFERENCES cane_sources (id),
    harvest_from   date NOT NULL,
    harvest_to     date NOT NULL,
    committed_tons numeric(18,3) NOT NULL DEFAULT 0,
    note           text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now(),
    created_by     text NOT NULL,
    updated_at     timestamptz NOT NULL DEFAULT now(),
    updated_by     text NOT NULL,
    row_version    bigint NOT NULL DEFAULT 1,
    CONSTRAINT cane_supply_key UNIQUE (version_id, source_id),
    CONSTRAINT cane_supply_window_ck CHECK (harvest_to >= harvest_from),
    CONSTRAINT cane_supply_nonneg_ck CHECK (committed_tons >= 0)
);

-- The delivery schedule: one row per source per day. PLAN is what the
-- generator scheduled, ACTUAL is what came through the gate, and the two never
-- overwrite each other - the same rule the rest of the plan follows.
CREATE TABLE daily_cane_supply (
    id            uuid PRIMARY KEY,
    version_id    uuid NOT NULL REFERENCES plan_versions (id) ON DELETE CASCADE,
    source_id     uuid NOT NULL REFERENCES cane_sources (id),
    factory_id    uuid NOT NULL REFERENCES factories (id),
    business_date date NOT NULL,
    series        text NOT NULL,
    tons          numeric(18,3) NOT NULL DEFAULT 0,
    trips         integer NOT NULL DEFAULT 0,
    pol_pct       numeric(6,3) NOT NULL DEFAULT 0,
    note          text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT daily_cane_supply_key UNIQUE (version_id, source_id, business_date, series),
    CONSTRAINT daily_cane_supply_series_ck CHECK (series IN ('PLAN','ACTUAL')),
    CONSTRAINT daily_cane_supply_nonneg_ck CHECK (tons >= 0 AND trips >= 0 AND pol_pct >= 0)
);
-- The two ways this table is read: a whole version's schedule by date, and one
-- source's deliveries across the season.
CREATE INDEX daily_cane_supply_date_idx ON daily_cane_supply (version_id, business_date, series);
CREATE INDEX daily_cane_supply_source_idx ON daily_cane_supply (source_id, business_date);
