-- Reverses 0005_security_governance.
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS export_jobs;
DROP TABLE IF EXISTS import_jobs;
DROP TABLE IF EXISTS outbox_events;
DROP RULE IF EXISTS audit_events_no_delete ON audit_events;
DROP RULE IF EXISTS audit_events_no_update ON audit_events;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS approval_steps;
DROP TABLE IF EXISTS approval_requests;
DROP TABLE IF EXISTS data_scopes;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS app_users;
