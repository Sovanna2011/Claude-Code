-- 0004_quality_downtime: laboratory results, quality holds, downtime and
-- maintenance windows.

CREATE TABLE quality_parameters (
    id           uuid PRIMARY KEY,
    code         text NOT NULL,
    name         text NOT NULL,
    uom          text NOT NULL DEFAULT '',
    test_method  text NOT NULL DEFAULT '',
    active       boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    created_by   text NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    updated_by   text NOT NULL,
    row_version  bigint NOT NULL DEFAULT 1,
    CONSTRAINT quality_parameters_code_key UNIQUE (code)
);

-- Specification limits are effective dated: tightening a colour limit must not
-- retrospectively fail last season's batches.
CREATE TABLE quality_specs (
    id            uuid PRIMARY KEY,
    product_id    uuid NOT NULL REFERENCES products (id),
    parameter_id  uuid NOT NULL REFERENCES quality_parameters (id),
    lower_limit   numeric(18,6),
    upper_limit   numeric(18,6),
    warn_lower    numeric(18,6),
    warn_upper    numeric(18,6),
    valid_from    date NOT NULL,
    valid_to      date,
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT quality_specs_key UNIQUE (product_id, parameter_id, valid_from),
    CONSTRAINT quality_specs_limits_ck CHECK (
        lower_limit IS NULL OR upper_limit IS NULL OR upper_limit >= lower_limit)
);

CREATE TABLE quality_samples (
    id            uuid PRIMARY KEY,
    sample_no     text NOT NULL,
    product_id    uuid NOT NULL REFERENCES products (id),
    batch_id      uuid REFERENCES batches (id),
    factory_id    uuid NOT NULL REFERENCES factories (id),
    business_date date NOT NULL,
    shift_id      uuid REFERENCES shifts (id),
    taken_at      timestamptz NOT NULL DEFAULT now(),
    lab_user      text NOT NULL DEFAULT '',
    status        text NOT NULL DEFAULT 'OPEN',
    comment       text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT quality_samples_no_key UNIQUE (sample_no),
    CONSTRAINT quality_samples_status_ck CHECK (status IN ('OPEN','COMPLETE','CANCELLED'))
);
CREATE INDEX quality_samples_date_idx ON quality_samples (factory_id, business_date);

CREATE TABLE quality_results (
    id           uuid PRIMARY KEY,
    sample_id    uuid NOT NULL REFERENCES quality_samples (id) ON DELETE CASCADE,
    parameter_id uuid NOT NULL REFERENCES quality_parameters (id),
    result_value numeric(18,6) NOT NULL,
    uom          text NOT NULL DEFAULT '',
    lower_limit  numeric(18,6),
    upper_limit  numeric(18,6),
    status       text NOT NULL,
    comment      text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    created_by   text NOT NULL,
    CONSTRAINT quality_results_key UNIQUE (sample_id, parameter_id),
    CONSTRAINT quality_results_status_ck CHECK (status IN ('PASS','WARNING','FAIL'))
);

-- A hold blocks shipment and consumption of the quantity it covers until an
-- authorised user releases it.
CREATE TABLE quality_holds (
    id            uuid PRIMARY KEY,
    warehouse_id  uuid NOT NULL REFERENCES warehouses (id),
    product_id    uuid NOT NULL REFERENCES products (id),
    batch_id      uuid REFERENCES batches (id),
    sample_id     uuid REFERENCES quality_samples (id),
    quantity      numeric(18,3) NOT NULL,
    placed_on     date NOT NULL,
    released_on   date,
    released_by   text,
    reason        text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT quality_holds_qty_ck CHECK (quantity > 0),
    CONSTRAINT quality_holds_release_ck CHECK (released_on IS NULL OR released_on >= placed_on)
);
CREATE INDEX quality_holds_open_idx ON quality_holds (warehouse_id, product_id)
    WHERE released_on IS NULL;

CREATE TABLE downtime_events (
    id               uuid PRIMARY KEY,
    factory_id       uuid NOT NULL REFERENCES factories (id),
    line_id          uuid REFERENCES production_lines (id),
    station_id       uuid REFERENCES stations (id),
    business_date    date NOT NULL,
    shift_id         uuid REFERENCES shifts (id),
    start_at         timestamptz NOT NULL,
    end_at           timestamptz NOT NULL,
    duration_hrs     numeric(8,3) NOT NULL,
    planned          boolean NOT NULL DEFAULT false,
    reason_code      text NOT NULL REFERENCES reason_codes (code),
    root_cause       text NOT NULL DEFAULT '',
    team             text NOT NULL DEFAULT '',
    corrective_action text NOT NULL DEFAULT '',
    created_at       timestamptz NOT NULL DEFAULT now(),
    created_by       text NOT NULL,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    updated_by       text NOT NULL,
    row_version      bigint NOT NULL DEFAULT 1,
    CONSTRAINT downtime_events_period_ck CHECK (end_at >= start_at),
    CONSTRAINT downtime_events_duration_ck CHECK (duration_hrs >= 0)
);
CREATE INDEX downtime_events_date_idx ON downtime_events (factory_id, business_date);

-- An approved maintenance window removes capacity from the plan: the generator
-- reads these as non-working days.
CREATE TABLE maintenance_windows (
    id            uuid PRIMARY KEY,
    factory_id    uuid NOT NULL REFERENCES factories (id),
    line_id       uuid REFERENCES production_lines (id),
    equipment_id  uuid REFERENCES equipment (id),
    start_date    date NOT NULL,
    end_date      date NOT NULL,
    description   text NOT NULL DEFAULT '',
    status        text NOT NULL DEFAULT 'PLANNED',
    approved_by   text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT maintenance_windows_period_ck CHECK (end_date >= start_date),
    CONSTRAINT maintenance_windows_status_ck CHECK (status IN ('PLANNED','APPROVED','DONE','CANCELLED'))
);
CREATE INDEX maintenance_windows_approved_idx ON maintenance_windows (factory_id, start_date)
    WHERE status = 'APPROVED';
