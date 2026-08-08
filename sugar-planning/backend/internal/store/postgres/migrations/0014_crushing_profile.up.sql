-- The shape of a crushing season.
--
-- The generator spread the cane target evenly across the campaign, and no mill
-- runs like that. The reference workbook - the mill's own daily plan for
-- 2026/27 - opens at 17,000 t while the boilers come up, settles at 19,000,
-- drops to half rate the day before each wash-out, stops for the wash-out, and
-- runs down through 15,000, 12,000, 8,000, 4,000 and 3,000 t as the cane ends.
--
-- Without somewhere to store that, regenerating a plan would flatten the mill's
-- own curve back into a straight line - and the date the raw silo fills, which
-- is the reason anyone plans jumbo bagging at all, would move.

CREATE TABLE plan_crushing_steps (
    id           uuid PRIMARY KEY,
    version_id   uuid NOT NULL REFERENCES plan_versions (id) ON DELETE CASCADE,
    kind         text NOT NULL,
    seq          integer NOT NULL,
    -- A RAMP_UP or RUN_DOWN step is a rate held for a number of days. A
    -- CLEANING step is a single campaign day the mill stops, so it carries the
    -- day number and no rate. The check constraint is what keeps the two from
    -- being written into each other's columns.
    rate_tons    numeric(14,3),
    days         integer,
    campaign_day integer,
    created_at   timestamptz NOT NULL DEFAULT now(),
    created_by   text NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    updated_by   text NOT NULL,
    row_version  bigint NOT NULL DEFAULT 1,
    CONSTRAINT plan_crushing_steps_key UNIQUE (version_id, kind, seq),
    CONSTRAINT plan_crushing_steps_kind_ck CHECK (kind IN ('RAMP_UP','RUN_DOWN','CLEANING')),
    CONSTRAINT plan_crushing_steps_shape_ck CHECK (
        (kind IN ('RAMP_UP','RUN_DOWN')
            AND rate_tons IS NOT NULL AND rate_tons >= 0
            AND days IS NOT NULL AND days > 0
            AND campaign_day IS NULL)
        OR
        (kind = 'CLEANING'
            AND campaign_day IS NOT NULL AND campaign_day > 0
            AND rate_tons IS NULL AND days IS NULL)
    )
);
CREATE INDEX plan_crushing_steps_version_idx ON plan_crushing_steps (version_id, kind, seq);
