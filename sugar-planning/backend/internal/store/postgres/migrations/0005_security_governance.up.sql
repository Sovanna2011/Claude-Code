-- 0005_security_governance: identity, approval workflow, audit and the
-- operational tables (outbox, jobs, idempotency, notifications).
--
-- Authentication itself is delegated to the enterprise identity provider. What
-- is stored here is the local projection of a user needed for authorisation,
-- data scoping and audit attribution. No password hashes are kept.

CREATE TABLE app_users (
    id            uuid PRIMARY KEY,
    subject       text NOT NULL,          -- OIDC subject claim
    username      text NOT NULL,
    display_name  text NOT NULL DEFAULT '',
    email         text NOT NULL DEFAULT '',
    active        boolean NOT NULL DEFAULT true,
    last_login_at timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    text NOT NULL,
    row_version   bigint NOT NULL DEFAULT 1,
    CONSTRAINT app_users_subject_key UNIQUE (subject),
    CONSTRAINT app_users_username_key UNIQUE (username)
);

CREATE TABLE roles (
    id          uuid PRIMARY KEY,
    code        text NOT NULL,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    created_by  text NOT NULL,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    updated_by  text NOT NULL,
    row_version bigint NOT NULL DEFAULT 1,
    CONSTRAINT roles_code_key UNIQUE (code)
);

CREATE TABLE permissions (
    code        text PRIMARY KEY,
    description text NOT NULL DEFAULT ''
);

CREATE TABLE role_permissions (
    role_id         uuid NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_code text NOT NULL REFERENCES permissions (code),
    PRIMARY KEY (role_id, permission_code)
);

CREATE TABLE user_roles (
    user_id     uuid NOT NULL REFERENCES app_users (id) ON DELETE CASCADE,
    role_id     uuid NOT NULL REFERENCES roles (id),
    granted_at  timestamptz NOT NULL DEFAULT now(),
    granted_by  text NOT NULL,
    PRIMARY KEY (user_id, role_id)
);

-- A data scope restricts a user to a company or a factory. Absence of any row
-- means "no access", not "all access": the service layer fails closed.
CREATE TABLE data_scopes (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES app_users (id) ON DELETE CASCADE,
    company_id uuid REFERENCES companies (id),
    factory_id uuid REFERENCES factories (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by text NOT NULL,
    CONSTRAINT data_scopes_target_ck CHECK (company_id IS NOT NULL OR factory_id IS NOT NULL)
);
CREATE INDEX data_scopes_user_idx ON data_scopes (user_id);

CREATE TABLE approval_requests (
    id            uuid PRIMARY KEY,
    entity        text NOT NULL,
    entity_id     uuid NOT NULL,
    action        text NOT NULL,
    requested_by  text NOT NULL,
    requested_at  timestamptz NOT NULL DEFAULT now(),
    status        text NOT NULL DEFAULT 'PENDING',
    decided_by    text,
    decided_at    timestamptz,
    comment       text NOT NULL DEFAULT '',
    CONSTRAINT approval_requests_status_ck CHECK (status IN ('PENDING','APPROVED','REJECTED','CANCELLED'))
);
CREATE INDEX approval_requests_pending_idx ON approval_requests (entity, entity_id)
    WHERE status = 'PENDING';

CREATE TABLE approval_steps (
    id          uuid PRIMARY KEY,
    request_id  uuid NOT NULL REFERENCES approval_requests (id) ON DELETE CASCADE,
    step_no     integer NOT NULL,
    approver    text NOT NULL,
    status      text NOT NULL DEFAULT 'PENDING',
    decided_at  timestamptz,
    comment     text NOT NULL DEFAULT '',
    CONSTRAINT approval_steps_key UNIQUE (request_id, step_no),
    CONSTRAINT approval_steps_status_ck CHECK (status IN ('PENDING','APPROVED','REJECTED','SKIPPED'))
);

CREATE TABLE comments (
    id          uuid PRIMARY KEY,
    entity      text NOT NULL,
    entity_id   uuid NOT NULL,
    body        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    created_by  text NOT NULL
);
CREATE INDEX comments_entity_idx ON comments (entity, entity_id);

-- --------------------------------------------------------------------------
-- Audit: append only for application users
-- --------------------------------------------------------------------------

CREATE TABLE audit_events (
    id             uuid PRIMARY KEY,
    occurred_at    timestamptz NOT NULL DEFAULT now(),
    actor          text NOT NULL,
    action         text NOT NULL,
    entity         text NOT NULL,
    entity_id      text NOT NULL DEFAULT '',
    before_state   jsonb,
    after_state    jsonb,
    reason         text NOT NULL DEFAULT '',
    correlation_id text NOT NULL DEFAULT '',
    source_ip      text NOT NULL DEFAULT ''
);
CREATE INDEX audit_events_entity_idx ON audit_events (entity, entity_id, occurred_at DESC);
CREATE INDEX audit_events_actor_idx ON audit_events (actor, occurred_at DESC);
CREATE INDEX audit_events_time_idx ON audit_events (occurred_at DESC);

-- The application role is granted INSERT and SELECT only; the revocation is
-- part of the deployment scripts rather than the schema, because the role name
-- is environment specific. A rule makes the intent explicit and stops an
-- accidental UPDATE from a migration or a console session.
CREATE RULE audit_events_no_update AS ON UPDATE TO audit_events DO INSTEAD NOTHING;
CREATE RULE audit_events_no_delete AS ON DELETE TO audit_events DO INSTEAD NOTHING;

-- --------------------------------------------------------------------------
-- Operational tables
-- --------------------------------------------------------------------------

-- Transactional outbox: integration events are written in the same transaction
-- as the business change and published afterwards, so an event can never
-- describe a change that was rolled back.
CREATE TABLE outbox_events (
    id             uuid PRIMARY KEY,
    topic          text NOT NULL,
    payload        jsonb NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    published_at   timestamptz,
    attempts       integer NOT NULL DEFAULT 0,
    last_error     text NOT NULL DEFAULT '',
    correlation_id text NOT NULL DEFAULT ''
);
CREATE INDEX outbox_events_unpublished_idx ON outbox_events (created_at)
    WHERE published_at IS NULL;

CREATE TABLE import_jobs (
    id            uuid PRIMARY KEY,
    kind          text NOT NULL,
    file_name     text NOT NULL,
    status        text NOT NULL DEFAULT 'PENDING',
    total_rows    integer NOT NULL DEFAULT 0,
    valid_rows    integer NOT NULL DEFAULT 0,
    error_rows    integer NOT NULL DEFAULT 0,
    errors        jsonb NOT NULL DEFAULT '[]'::jsonb,
    committed_at  timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    text NOT NULL,
    CONSTRAINT import_jobs_status_ck CHECK (status IN
        ('PENDING','VALIDATED','COMMITTED','FAILED','CANCELLED'))
);

CREATE TABLE export_jobs (
    id           uuid PRIMARY KEY,
    report_code  text NOT NULL,
    format       text NOT NULL,
    parameters   jsonb NOT NULL DEFAULT '{}'::jsonb,
    status       text NOT NULL DEFAULT 'PENDING',
    file_name    text NOT NULL DEFAULT '',
    size_bytes   bigint NOT NULL DEFAULT 0,
    completed_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    created_by   text NOT NULL,
    CONSTRAINT export_jobs_format_ck CHECK (format IN ('CSV','XLSX','PDF')),
    CONSTRAINT export_jobs_status_ck CHECK (status IN ('PENDING','RUNNING','DONE','FAILED'))
);

CREATE TABLE attachments (
    id          uuid PRIMARY KEY,
    entity      text NOT NULL,
    entity_id   uuid NOT NULL,
    file_name   text NOT NULL,
    mime_type   text NOT NULL,
    size_bytes  bigint NOT NULL,
    storage_key text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    created_by  text NOT NULL,
    CONSTRAINT attachments_size_ck CHECK (size_bytes >= 0)
);

CREATE TABLE notifications (
    id         uuid PRIMARY KEY,
    recipient  text NOT NULL,
    severity   text NOT NULL DEFAULT 'INFO',
    code       text NOT NULL,
    title      text NOT NULL,
    detail     text NOT NULL DEFAULT '',
    entity     text NOT NULL DEFAULT '',
    entity_id  text NOT NULL DEFAULT '',
    read_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT notifications_severity_ck CHECK (severity IN ('INFO','SUCCESS','WARNING','ERROR'))
);
CREATE INDEX notifications_unread_idx ON notifications (recipient, created_at DESC)
    WHERE read_at IS NULL;

-- Idempotency keys make a retried posting safe: the second call returns the
-- first call's response instead of posting twice.
CREATE TABLE idempotency_keys (
    key         text NOT NULL,
    endpoint    text NOT NULL,
    response    bytea,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (endpoint, key)
);
CREATE INDEX idempotency_keys_created_idx ON idempotency_keys (created_at);
