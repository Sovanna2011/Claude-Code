-- 0007_costing: cost elements, effective-dated rates, exchange rates and saved
-- cost runs.
--
-- A cost is a rate multiplied by a driver quantity. Keeping the two apart in
-- the schema is what makes the variance decomposable later: when a cost moves,
-- it moved because the rate changed or because the quantity changed, and a
-- controller needs to be able to say which.

CREATE TABLE cost_elements (
    id           uuid PRIMARY KEY,
    code         text NOT NULL,
    name         text NOT NULL,
    category     text NOT NULL,
    -- What the element scales with. A fixed seasonal cost has a driver
    -- quantity of one, so the same rate x quantity arithmetic covers it.
    driver       text NOT NULL,
    -- Whether the site treats it as variable. Deliberately not derived from
    -- the driver: a shift crew paid per run hour may still be a fixed cost to
    -- a site with a salaried crew, and the contribution analysis has to follow
    -- the site's view rather than the software's.
    variable     boolean NOT NULL DEFAULT true,
    note         text NOT NULL DEFAULT '',
    valid_from   date,
    valid_to     date,
    active       boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    created_by   text NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    updated_by   text NOT NULL,
    row_version  bigint NOT NULL DEFAULT 1,
    CONSTRAINT cost_elements_code_key UNIQUE (code),
    CONSTRAINT cost_elements_category_ck CHECK (category IN
        ('CANE','LABOUR','ENERGY','CHEMICALS','PACKAGING','MAINTENANCE','OVERHEAD')),
    CONSTRAINT cost_elements_driver_ck CHECK (driver IN
        ('CANE_TON','SUGAR_TON','RUN_HOUR','CALENDAR_DAY','FIXED_SEASON'))
);

-- Rates are effective dated so that a mid-season fuel price rise does not
-- rewrite the cost of the weeks before it.
CREATE TABLE cost_rates (
    id          uuid PRIMARY KEY,
    element_id  uuid NOT NULL REFERENCES cost_elements (id),
    factory_id  uuid NOT NULL REFERENCES factories (id),
    season_id   uuid REFERENCES seasons (id),
    rate_type   text NOT NULL,
    rate        numeric(18,6) NOT NULL,
    currency    text NOT NULL,
    valid_from  date NOT NULL,
    valid_to    date,
    note        text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    created_by  text NOT NULL,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    updated_by  text NOT NULL,
    row_version bigint NOT NULL DEFAULT 1,
    CONSTRAINT cost_rates_key UNIQUE (element_id, factory_id, rate_type, valid_from),
    CONSTRAINT cost_rates_type_ck CHECK (rate_type IN ('STANDARD','ACTUAL')),
    CONSTRAINT cost_rates_period_ck CHECK (valid_to IS NULL OR valid_to >= valid_from),
    CONSTRAINT cost_rates_currency_ck CHECK (char_length(currency) = 3)
);
CREATE INDEX cost_rates_lookup_idx ON cost_rates (factory_id, rate_type, valid_from);

-- Cane is paid in riel, fuel is invoiced in dollars, and the board reports in
-- one of them. Only one direction of a pair need be entered; the calculation
-- uses the inverse when the direct quotation is absent.
CREATE TABLE exchange_rates (
    id            uuid PRIMARY KEY,
    from_currency text NOT NULL,
    to_currency   text NOT NULL,
    rate          numeric(18,6) NOT NULL,
    valid_from    date NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT exchange_rates_key UNIQUE (from_currency, to_currency, valid_from),
    CONSTRAINT exchange_rates_rate_ck CHECK (rate > 0),
    CONSTRAINT exchange_rates_pair_ck CHECK (from_currency <> to_currency),
    CONSTRAINT exchange_rates_currency_ck CHECK (
        char_length(from_currency) = 3 AND char_length(to_currency) = 3)
);

-- A saved run, so that a figure quoted in a board pack can be reproduced later
-- even after the rates have moved on. The lines are stored rather than
-- recomputed for the same reason.
CREATE TABLE cost_runs (
    id            uuid PRIMARY KEY,
    season_id     uuid NOT NULL REFERENCES seasons (id),
    version_id    uuid REFERENCES plan_versions (id),
    factory_id    uuid NOT NULL REFERENCES factories (id),
    code          text NOT NULL,
    from_date     date NOT NULL,
    to_date       date NOT NULL,
    currency      text NOT NULL,
    totals        jsonb NOT NULL DEFAULT '{}'::jsonb,
    note          text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT cost_runs_code_key UNIQUE (season_id, code),
    CONSTRAINT cost_runs_period_ck CHECK (to_date >= from_date)
);
CREATE INDEX cost_runs_season_idx ON cost_runs (season_id, from_date);

CREATE TABLE cost_run_lines (
    id             uuid PRIMARY KEY,
    run_id         uuid NOT NULL REFERENCES cost_runs (id) ON DELETE CASCADE,
    element_id     uuid NOT NULL REFERENCES cost_elements (id),
    standard_rate  numeric(18,6) NOT NULL DEFAULT 0,
    actual_rate    numeric(18,6) NOT NULL DEFAULT 0,
    planned_qty    numeric(18,3) NOT NULL DEFAULT 0,
    actual_qty     numeric(18,3) NOT NULL DEFAULT 0,
    planned_cost   numeric(18,2) NOT NULL DEFAULT 0,
    actual_cost    numeric(18,2) NOT NULL DEFAULT 0,
    rate_variance  numeric(18,2) NOT NULL DEFAULT 0,
    usage_variance numeric(18,2) NOT NULL DEFAULT 0,
    total_variance numeric(18,2) NOT NULL DEFAULT 0,
    missing        text NOT NULL DEFAULT '',
    CONSTRAINT cost_run_lines_key UNIQUE (run_id, element_id)
);
