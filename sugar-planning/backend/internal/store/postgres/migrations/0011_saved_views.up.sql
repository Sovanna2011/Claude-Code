-- 0011_saved_views: the filter somebody set up last Tuesday.
--
-- Section 15 asks for variant management, saved views and personalization. All
-- three are the same thing stored: a named set of filters, sorts and column
-- choices that a page can be put back into.
--
-- Two decisions worth recording.
--
-- The payload is jsonb and deliberately not modelled. What a view holds is a
-- property of the screen it belongs to - the planning board saves a date range
-- and a series, the order list saves a status filter - and a table with a column
-- per filter would need migrating every time a screen grew one. The server does
-- not interpret it; it stores what the page sent and hands it back.
--
-- A view belongs to the person who made it. Shared views exist because a factory
-- that has worked out the right filter for a morning review should not each
-- rediscover it, but only the owner may change or delete one: a variant that
-- anybody can edit is a variant nobody can rely on.
CREATE TABLE saved_views (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- owner is the username from the token, not a foreign key: there is no user
    -- table here, because identity lives in the identity provider.
    owner        text        NOT NULL,
    -- page is the route name, so a view cannot be offered on a screen that
    -- cannot apply it.
    page         text        NOT NULL,
    name         text        NOT NULL,
    -- factory_id scopes a shared view. A personal view carries it too, so that
    -- somebody working across two factories does not see the other one's
    -- filters offered on this one's screen.
    factory_id   uuid        REFERENCES factories (id),
    shared       boolean     NOT NULL DEFAULT false,
    -- is_default applies the view on arrival. At most one per owner and page,
    -- enforced by the partial unique index below rather than by application
    -- code that two tabs could race.
    is_default   boolean     NOT NULL DEFAULT false,
    payload      jsonb       NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    created_by   text        NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    updated_by   text        NOT NULL,
    row_version  bigint      NOT NULL DEFAULT 1,

    CONSTRAINT saved_views_name_not_blank CHECK (btrim(name) <> '')
);

-- One view of a given name per person per page: saving over a name replaces it,
-- which is what "save" on a variant means.
CREATE UNIQUE INDEX saved_views_name_idx
    ON saved_views (owner, page, lower(name));

-- At most one default per person per page.
CREATE UNIQUE INDEX saved_views_default_idx
    ON saved_views (owner, page)
    WHERE is_default;

-- How the list is actually read: what may I see on this page.
CREATE INDEX saved_views_page_idx
    ON saved_views (page, owner, shared, factory_id);
