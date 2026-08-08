DROP INDEX IF EXISTS notifications_unread_idx;
ALTER TABLE notifications DROP COLUMN IF EXISTS factory_id;
CREATE INDEX notifications_unread_idx ON notifications (recipient, created_at DESC)
    WHERE read_at IS NULL;
