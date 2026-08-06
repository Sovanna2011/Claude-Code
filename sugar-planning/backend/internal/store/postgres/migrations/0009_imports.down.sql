DROP TABLE IF EXISTS import_rows;
ALTER TABLE import_jobs
    DROP COLUMN IF EXISTS mapping_id,
    DROP COLUMN IF EXISTS version_id,
    DROP COLUMN IF EXISTS factory_id,
    DROP COLUMN IF EXISTS kind_note,
    DROP COLUMN IF EXISTS committed_by;
DROP TABLE IF EXISTS import_mappings;
