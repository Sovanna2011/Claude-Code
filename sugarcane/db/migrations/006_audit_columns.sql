-- Who created a row and when, and who last changed it and when — on every business table, and
-- stamped by the database rather than by whatever asked it to write.
--
-- Six tables already carried these four columns; nine did not. Rather than introduce a second
-- naming convention this migration extends the existing one, so every table reads the same way.
--
-- The times come from the database clock. A caller cannot supply one, cannot backdate a row and
-- cannot change a creation stamp afterwards: the trigger below overwrites whatever arrived. That
-- is the point of the requirement — an audit column a client can set is not an audit column.

ALTER TABLE company              ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN created_by text        NOT NULL DEFAULT 'system',
                                 ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN updated_by text        NOT NULL DEFAULT 'system';

ALTER TABLE plantation           ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN created_by text        NOT NULL DEFAULT 'system',
                                 ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN updated_by text        NOT NULL DEFAULT 'system';

ALTER TABLE non_plantable_reason ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN created_by text        NOT NULL DEFAULT 'system',
                                 ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN updated_by text        NOT NULL DEFAULT 'system';

ALTER TABLE block_non_plantable  ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN created_by text        NOT NULL DEFAULT 'system',
                                 ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN updated_by text        NOT NULL DEFAULT 'system';

ALTER TABLE app_user             ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN created_by text        NOT NULL DEFAULT 'system',
                                 ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN updated_by text        NOT NULL DEFAULT 'system';

ALTER TABLE crop_season          ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN created_by text        NOT NULL DEFAULT 'system',
                                 ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN updated_by text        NOT NULL DEFAULT 'system';

ALTER TABLE cane_variety         ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN created_by text        NOT NULL DEFAULT 'system',
                                 ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN updated_by text        NOT NULL DEFAULT 'system';

ALTER TABLE activity_dependency  ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN created_by text        NOT NULL DEFAULT 'system',
                                 ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN updated_by text        NOT NULL DEFAULT 'system';

ALTER TABLE projection_line      ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN created_by text        NOT NULL DEFAULT 'system',
                                 ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
                                 ADD COLUMN updated_by text        NOT NULL DEFAULT 'system';

-- ---------------------------------------------------------------- the stamp
--
-- One function for every table. It answers three questions the same way everywhere:
--
--   when  — always now(), the database's own clock. Never a value the caller sent, so rows cannot
--           be backdated and two servers with drifting clocks still agree.
--   who   — the signed-in user, taken from the transaction-local setting the API sets on every
--           transaction. A write that arrives without one (a migration, a fix applied with psql)
--           is stamped 'system' rather than left blank.
--   never — the creation stamp is copied from the old row on every update, so nothing can rewrite
--           who created a record or when.

CREATE OR REPLACE FUNCTION stamp_row_change() RETURNS trigger AS $$
DECLARE
    actor text := NULLIF(current_setting('app.actor', true), '');
BEGIN
    IF TG_OP = 'INSERT' THEN
        NEW.created_at := now();
        NEW.updated_at := now();
        NEW.created_by := COALESCE(actor, NULLIF(NEW.created_by, ''), 'system');
        NEW.updated_by := NEW.created_by;
    ELSE
        -- Immutable, whatever the UPDATE statement said.
        NEW.created_at := OLD.created_at;
        NEW.created_by := OLD.created_by;
        NEW.updated_at := now();
        NEW.updated_by := COALESCE(actor, NULLIF(NEW.updated_by, ''), OLD.updated_by, 'system');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attached to every table that carries the four columns, including the six that already had them —
-- those were stamped by hand in each statement, which only holds while every statement remembers.
DO $$
DECLARE
    target text;
BEGIN
    FOR target IN
        SELECT c.relname
          FROM pg_class c
          JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = 'public'
         WHERE c.relkind = 'r'
           AND EXISTS (SELECT 1 FROM pg_attribute a
                        WHERE a.attrelid = c.oid AND a.attname = 'created_at' AND NOT a.attisdropped)
           AND EXISTS (SELECT 1 FROM pg_attribute a
                        WHERE a.attrelid = c.oid AND a.attname = 'updated_by' AND NOT a.attisdropped)
         ORDER BY c.relname
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I', 'stamp_' || target, target);
        EXECUTE format(
            'CREATE TRIGGER %I BEFORE INSERT OR UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION stamp_row_change()',
            'stamp_' || target, target);
    END LOOP;
END;
$$;

-- The audit trail itself is deliberately left alone. audit_log and projection_approval are
-- append-only and already record the same two facts under the names their readers and indexes use:
-- `at` is when, `actor` is who. Adding a second pair would give each row two creation stamps that
-- could disagree.

INSERT INTO schema_migrations(version) VALUES ('006_audit_columns');
