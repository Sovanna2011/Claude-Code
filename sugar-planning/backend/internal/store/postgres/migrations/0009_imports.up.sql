-- 0009_imports: the controlled mapping template and the staging area behind it.
--
-- A file is never posted as it arrives. It is parsed into import_rows,
-- validated row by row, shown to somebody, and committed only when they say so.
-- An import that wrote straight through would be a way of getting past every
-- rule the rest of the system enforces - and the spreadsheet this application
-- replaces is exactly where the bad data comes from.

CREATE TABLE import_mappings (
    id             uuid PRIMARY KEY,
    code           text NOT NULL,
    name           text NOT NULL,
    kind           text NOT NULL,
    -- The column mapping is jsonb because its shape is the mapping itself:
    -- a list of field-to-column pairs that varies per site and per file. A
    -- table of columns would buy nothing - nothing joins to it, and nothing
    -- queries inside it.
    columns        jsonb NOT NULL DEFAULT '[]'::jsonb,
    header_row     integer NOT NULL DEFAULT 1,
    first_data_row integer NOT NULL DEFAULT 0,
    delimiter      text NOT NULL DEFAULT '',
    date_format    text NOT NULL DEFAULT '',
    decimal_comma  boolean NOT NULL DEFAULT false,
    note           text NOT NULL DEFAULT '',
    valid_from     date,
    valid_to       date,
    active         boolean NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now(),
    created_by     text NOT NULL,
    updated_at     timestamptz NOT NULL DEFAULT now(),
    updated_by     text NOT NULL,
    row_version    bigint NOT NULL DEFAULT 1,
    CONSTRAINT import_mappings_code_key UNIQUE (code),
    CONSTRAINT import_mappings_kind_ck CHECK (kind IN
        ('DAILY_CANE','DAILY_PRODUCTION','DAILY_SHIPMENT','DAILY_STORAGE')),
    CONSTRAINT import_mappings_rows_ck CHECK (header_row >= 0 AND first_data_row >= 0)
);

-- The job gains what it needs to be resumed and committed: which mapping read
-- it, which plan version it is going into, and which factory.
ALTER TABLE import_jobs
    ADD COLUMN mapping_id uuid REFERENCES import_mappings (id),
    ADD COLUMN version_id uuid REFERENCES plan_versions (id),
    ADD COLUMN factory_id uuid REFERENCES factories (id),
    ADD COLUMN kind_note  text NOT NULL DEFAULT '',
    ADD COLUMN committed_by text NOT NULL DEFAULT '';

CREATE TABLE import_rows (
    id         uuid PRIMARY KEY,
    job_id     uuid NOT NULL REFERENCES import_jobs (id) ON DELETE CASCADE,
    -- row_no is the line in the file, one-based, so an error names what
    -- somebody sees in Excel rather than an offset into an array.
    row_no     integer NOT NULL,
    values     jsonb NOT NULL DEFAULT '{}'::jsonb,
    errors     jsonb NOT NULL DEFAULT '[]'::jsonb,
    duplicate  boolean NOT NULL DEFAULT false,
    replaces   boolean NOT NULL DEFAULT false,
    CONSTRAINT import_rows_key UNIQUE (job_id, row_no)
);
CREATE INDEX import_rows_job_idx ON import_rows (job_id, row_no);
