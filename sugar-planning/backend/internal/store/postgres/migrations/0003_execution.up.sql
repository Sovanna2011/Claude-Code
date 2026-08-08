-- 0003_execution: production orders, confirmations and inventory documents.
--
-- Inventory is document based. A balance is never edited directly: it is the
-- consequence of posted documents, and a mistake is corrected with a reversal
-- document that points back at the original. That is what makes the stock
-- ledger auditable and what lets a confirmation be undone without deleting
-- history.

CREATE TABLE batches (
    id            uuid PRIMARY KEY,
    code          text NOT NULL,
    product_id    uuid NOT NULL REFERENCES products (id),
    factory_id    uuid NOT NULL REFERENCES factories (id),
    produced_on   date,
    quantity      numeric(18,3) NOT NULL DEFAULT 0,
    status        text NOT NULL DEFAULT 'OPEN',
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT batches_code_key UNIQUE (code),
    CONSTRAINT batches_qty_ck CHECK (quantity >= 0),
    CONSTRAINT batches_status_ck CHECK (status IN ('OPEN','RELEASED','BLOCKED','CONSUMED'))
);

CREATE TABLE production_orders (
    id              uuid PRIMARY KEY,
    order_no        text NOT NULL,
    company_id      uuid NOT NULL REFERENCES companies (id),
    factory_id      uuid NOT NULL REFERENCES factories (id),
    line_id         uuid REFERENCES production_lines (id),
    version_id      uuid REFERENCES plan_versions (id),
    business_date   date NOT NULL,
    shift_id        uuid REFERENCES shifts (id),
    product_id      uuid NOT NULL REFERENCES products (id),
    packaging_id    uuid REFERENCES packaging_types (id),
    planned_qty     numeric(18,3) NOT NULL DEFAULT 0,
    confirmed_qty   numeric(18,3) NOT NULL DEFAULT 0,
    batch_id        uuid REFERENCES batches (id),
    bom_version     text NOT NULL DEFAULT '',
    planned_start   timestamptz,
    planned_end     timestamptz,
    priority        integer NOT NULL DEFAULT 5,
    status          text NOT NULL DEFAULT 'PLANNED',
    team            text NOT NULL DEFAULT '',
    variance_reason text REFERENCES reason_codes (code),
    created_at      timestamptz NOT NULL DEFAULT now(),
    created_by      text NOT NULL,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    updated_by      text NOT NULL,
    row_version     bigint NOT NULL DEFAULT 1,
    CONSTRAINT production_orders_no_key UNIQUE (order_no),
    CONSTRAINT production_orders_qty_ck CHECK (planned_qty >= 0 AND confirmed_qty >= 0),
    CONSTRAINT production_orders_priority_ck CHECK (priority BETWEEN 1 AND 9),
    CONSTRAINT production_orders_status_ck CHECK (status IN
        ('PLANNED','RELEASED','IN_PROCESS','PARTIALLY_CONFIRMED','COMPLETED',
         'TECHNICALLY_CLOSED','CANCELLED')),
    CONSTRAINT production_orders_dates_ck CHECK (planned_end IS NULL OR planned_start IS NULL
        OR planned_end >= planned_start)
);
CREATE INDEX production_orders_open_idx ON production_orders (factory_id, business_date)
    WHERE status IN ('PLANNED','RELEASED','IN_PROCESS','PARTIALLY_CONFIRMED');

CREATE TABLE production_confirmations (
    id               uuid PRIMARY KEY,
    order_id         uuid NOT NULL REFERENCES production_orders (id),
    confirmation_no  text NOT NULL,
    business_date    date NOT NULL,
    shift_id         uuid REFERENCES shifts (id),
    yield_qty        numeric(18,3) NOT NULL DEFAULT 0,
    scrap_qty        numeric(18,3) NOT NULL DEFAULT 0,
    rework_qty       numeric(18,3) NOT NULL DEFAULT 0,
    labour_hours     numeric(10,2) NOT NULL DEFAULT 0,
    machine_hours    numeric(10,2) NOT NULL DEFAULT 0,
    batch_id         uuid REFERENCES batches (id),
    reversal_of      uuid REFERENCES production_confirmations (id),
    reversed         boolean NOT NULL DEFAULT false,
    reason_code      text REFERENCES reason_codes (code),
    created_at       timestamptz NOT NULL DEFAULT now(),
    created_by       text NOT NULL,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    updated_by       text NOT NULL,
    row_version      bigint NOT NULL DEFAULT 1,
    CONSTRAINT production_confirmations_no_key UNIQUE (confirmation_no),
    CONSTRAINT production_confirmations_qty_ck CHECK (
        yield_qty >= 0 AND scrap_qty >= 0 AND rework_qty >= 0),
    CONSTRAINT production_confirmations_hours_ck CHECK (labour_hours >= 0 AND machine_hours >= 0)
);
CREATE INDEX production_confirmations_order_idx ON production_confirmations (order_id);

CREATE TABLE material_consumptions (
    id              uuid PRIMARY KEY,
    confirmation_id uuid NOT NULL REFERENCES production_confirmations (id),
    material_id     uuid NOT NULL REFERENCES materials (id),
    quantity        numeric(18,3) NOT NULL,
    uom             text NOT NULL REFERENCES units_of_measure (code),
    created_at      timestamptz NOT NULL DEFAULT now(),
    created_by      text NOT NULL,
    CONSTRAINT material_consumptions_key UNIQUE (confirmation_id, material_id)
);

-- --------------------------------------------------------------------------
-- Inventory documents
-- --------------------------------------------------------------------------

CREATE TABLE inventory_documents (
    id             uuid PRIMARY KEY,
    document_no    text NOT NULL,
    doc_type       text NOT NULL,
    business_date  date NOT NULL,
    posted_at      timestamptz NOT NULL DEFAULT now(),
    factory_id     uuid NOT NULL REFERENCES factories (id),
    reference      text NOT NULL DEFAULT '',
    reversal_of    uuid REFERENCES inventory_documents (id),
    reversed       boolean NOT NULL DEFAULT false,
    reason_code    text REFERENCES reason_codes (code),
    note           text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now(),
    created_by     text NOT NULL,
    CONSTRAINT inventory_documents_no_key UNIQUE (document_no),
    CONSTRAINT inventory_documents_type_ck CHECK (doc_type IN
        ('RECEIPT','ISSUE','TRANSFER','ADJUSTMENT','COUNT','HOLD','RELEASE','SHIPMENT','REVERSAL'))
);
CREATE INDEX inventory_documents_date_idx ON inventory_documents (factory_id, business_date);

CREATE TABLE inventory_document_items (
    id             uuid PRIMARY KEY,
    document_id    uuid NOT NULL REFERENCES inventory_documents (id) ON DELETE CASCADE,
    line_no        integer NOT NULL,
    warehouse_id   uuid NOT NULL REFERENCES warehouses (id),
    product_id     uuid NOT NULL REFERENCES products (id),
    batch_id       uuid REFERENCES batches (id),
    -- Signed: a receipt is positive, an issue is negative. The sign lives in
    -- the item, so a reversal is simply the same items with the sign flipped.
    quantity       numeric(18,3) NOT NULL,
    uom            text NOT NULL REFERENCES units_of_measure (code),
    to_warehouse   uuid REFERENCES warehouses (id),
    created_at     timestamptz NOT NULL DEFAULT now(),
    created_by     text NOT NULL,
    CONSTRAINT inventory_document_items_key UNIQUE (document_id, line_no),
    CONSTRAINT inventory_document_items_qty_ck CHECK (quantity <> 0)
);
CREATE INDEX inventory_document_items_stock_idx ON inventory_document_items (warehouse_id, product_id);

-- Denormalised current balance, maintained inside the same transaction as the
-- documents that move it. It is a cache of the document history, never the
-- source of truth.
CREATE TABLE stock_balances (
    warehouse_id  uuid NOT NULL REFERENCES warehouses (id),
    product_id    uuid NOT NULL REFERENCES products (id),
    quantity      numeric(18,3) NOT NULL DEFAULT 0,
    hold_quantity numeric(18,3) NOT NULL DEFAULT 0,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    PRIMARY KEY (warehouse_id, product_id),
    CONSTRAINT stock_balances_hold_ck CHECK (hold_quantity >= 0 AND hold_quantity <= quantity)
);

CREATE TABLE stock_reservations (
    id           uuid PRIMARY KEY,
    warehouse_id uuid NOT NULL REFERENCES warehouses (id),
    product_id   uuid NOT NULL REFERENCES products (id),
    quantity     numeric(18,3) NOT NULL,
    reference    text NOT NULL,
    valid_until  date,
    created_at   timestamptz NOT NULL DEFAULT now(),
    created_by   text NOT NULL,
    CONSTRAINT stock_reservations_qty_ck CHECK (quantity > 0)
);

CREATE TABLE stock_counts (
    id            uuid PRIMARY KEY,
    document_id   uuid REFERENCES inventory_documents (id),
    warehouse_id  uuid NOT NULL REFERENCES warehouses (id),
    product_id    uuid NOT NULL REFERENCES products (id),
    business_date date NOT NULL,
    counted_qty   numeric(18,3) NOT NULL,
    system_qty    numeric(18,3) NOT NULL,
    difference    numeric(18,3) NOT NULL,
    reason_code   text REFERENCES reason_codes (code),
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    CONSTRAINT stock_counts_key UNIQUE (warehouse_id, product_id, business_date),
    CONSTRAINT stock_counts_qty_ck CHECK (counted_qty >= 0)
);
