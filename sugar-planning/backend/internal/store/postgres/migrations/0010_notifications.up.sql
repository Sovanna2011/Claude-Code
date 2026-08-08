-- 0010_notifications: address the inbox the way this system knows people.
--
-- There is no user directory here: identity, roles and data scope all come from
-- the token the identity provider issued. So a notification is addressed to a
-- **role** at a **factory** rather than to a named person - "the shipment
-- planners at Kampong Speu" - and somebody's inbox is what their roles and their
-- scope entitle them to see. That is also the better answer operationally: a
-- capacity warning addressed to a person who has left is a warning nobody owns.
ALTER TABLE notifications
    ADD COLUMN factory_id uuid REFERENCES factories (id);

-- The recipient column now holds a role code. The index follows how the inbox is
-- actually read: my roles, unread first.
DROP INDEX IF EXISTS notifications_unread_idx;
CREATE INDEX notifications_unread_idx ON notifications (recipient, factory_id, created_at DESC)
    WHERE read_at IS NULL;
