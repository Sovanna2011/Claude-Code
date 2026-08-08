-- Farm Area Monitoring — sample plantation
--
-- One company with a plantation of three farms near Lusaka. Block boundaries are generated as
-- real rectangles and the stored total area is read back out of PostGIS with ST_Area, so the
-- figure on the dashboard and the polygon on the map are the same fact rather than two facts that
-- happen to agree. Zone and farm boundaries are the union of what is beneath them.
--
-- One zone is deliberately left with no blocks — the specification's own example has one — so the
-- roll-up is exercised against an empty branch rather than only against tidy data.

INSERT INTO company(code, name) VALUES ('SGC', 'Sugarcane Group');
INSERT INTO plantation(company_id, code, name)
    SELECT id, 'NP', 'Northern Plantation' FROM company WHERE code = 'SGC';

INSERT INTO crop_season(code, name, crop_year, starts_on, ends_on) VALUES
    ('CS2025', 'Season 2025 / 2026', 2025, '2025-03-01', '2026-02-28'),
    ('CS2026', 'Season 2026 / 2027', 2026, '2026-03-01', '2027-02-28');

INSERT INTO cane_variety(code, name, growing_period_months) VALUES
    ('CO0238',  'CO 0238',     12),
    ('NCO376',  'NCo 376',     14),
    ('CP881762','CP 88-1762',  13);

INSERT INTO non_plantable_reason(code, name, sort_order) VALUES
    ('ROAD',      'Road',                    10),
    ('CANAL',     'Canal',                   20),
    ('POND',      'Pond',                    30),
    ('BUILDING',  'Building',                40),
    ('MOUNTAIN',  'Mountain',                50),
    ('FOREST',    'Forest / protected area', 60),
    ('FLOOD',     'Flooded area',            70),
    ('INFRA',     'Infrastructure',          80),
    ('RESERVED',  'Reserved land',           90),
    ('OTHER',     'Other unusable area',    100);

-- Password for every demo account is Farm#2026.
INSERT INTO app_user(username, password_hash, full_name, role) VALUES
    ('admin',   '$2a$10$7sk2BBBwARnT/j2KUn82bO5LoPcTODpx5fQXhjbED3JcqlNFOdTOe', 'System Administrator', 'Admin'),
    ('manager', '$2a$10$7sk2BBBwARnT/j2KUn82bO5LoPcTODpx5fQXhjbED3JcqlNFOdTOe', 'Plantation Manager',   'Manager'),
    ('planner', '$2a$10$7sk2BBBwARnT/j2KUn82bO5LoPcTODpx5fQXhjbED3JcqlNFOdTOe', 'Agricultural Planner', 'Planner'),
    ('viewer',  '$2a$10$7sk2BBBwARnT/j2KUn82bO5LoPcTODpx5fQXhjbED3JcqlNFOdTOe', 'Report Viewer',        'Viewer');

DO $seed$
DECLARE
    plantation_id  integer;
    farm_id        integer;
    zone_id        integer;
    block_id       integer;

    farms          text[][] := ARRAY[
        ARRAY['FRM-01', 'Riverside Farm', 'J. Almeida',  '-15.474', '28.232'],
        ARRAY['FRM-02', 'Highland Farm',  'P. Okonkwo',  '-15.398', '28.318'],
        ARRAY['FRM-03', 'Valley Farm',    'S. Ramirez',  '-15.322', '28.404']
    ];
    -- zones per farm, and blocks per zone; the third zone of Riverside has none
    zone_plan      integer[][] := ARRAY[ARRAY[4, 3, 0], ARRAY[4, 3, 0], ARRAY[3, 3, 0]];

    f              integer;
    z              integer;
    k              integer;
    zones_here     integer;
    blocks_here    integer;

    origin_lat     numeric;
    origin_lng     numeric;
    lat0           numeric;
    lng0           numeric;
    side           numeric := 0.0105;   -- ≈ 1.16 km, a plausible field
    gap            numeric := 0.0012;

    total_ha       numeric(12,4);
    plantable_ha   numeric(12,4);
    unplantable    numeric(12,4);
    block_no       integer := 0;
    zone_no        integer := 0;

    season_2025    integer;
    season_2026    integer;
    variety_ids    integer[];
BEGIN
    SELECT p.id INTO plantation_id FROM plantation p JOIN company c ON c.id = p.company_id WHERE c.code = 'SGC';
    SELECT id INTO season_2025 FROM crop_season WHERE code = 'CS2025';
    SELECT id INTO season_2026 FROM crop_season WHERE code = 'CS2026';
    SELECT array_agg(id ORDER BY id) INTO variety_ids FROM cane_variety;

    FOR f IN 1 .. array_length(farms, 1) LOOP
        INSERT INTO farm(plantation_id, code, name, manager_name)
             VALUES (plantation_id, farms[f][1], farms[f][2], farms[f][3])
          RETURNING id INTO farm_id;

        origin_lat := farms[f][4]::numeric;
        origin_lng := farms[f][5]::numeric;
        zones_here := 0;
        FOR z IN 1 .. 3 LOOP
            IF zone_plan[f][z] IS NULL THEN CONTINUE; END IF;
            zones_here := zones_here + 1;
            zone_no := zone_no + 1;

            INSERT INTO zone(farm_id, code, name, supervisor_name)
                 VALUES (farm_id,
                         farms[f][1] || '-Z' || z,
                         farms[f][2] || ' Zone ' || z,
                         CASE WHEN z = 1 THEN 'M. Diallo' WHEN z = 2 THEN 'T. Nakamura' ELSE 'A. Banda' END)
              RETURNING id INTO zone_id;

            blocks_here := zone_plan[f][z];
            FOR k IN 1 .. GREATEST(blocks_here, 0) LOOP
                block_no := block_no + 1;

                -- zones march south-east across the farm; blocks fill a two-column grid
                lat0 := origin_lat - (z - 1) * 0.026 - ((k - 1) / 2) * (side + gap);
                lng0 := origin_lng + (z - 1) * 0.030 + ((k - 1) % 2) * (side + gap);

                INSERT INTO block(zone_id, code, name, total_area_ha, plantable_area_ha,
                                  latitude, longitude, boundary, land_status, cane_status)
                VALUES (
                    zone_id,
                    'BLK-' || lpad(block_no::text, 3, '0'),
                    farms[f][2] || ' Zone ' || z || ' Block ' || k,
                    0, 0,                                  -- replaced below from the polygon
                    round(lat0 - side / 2, 6),
                    round(lng0 + side / 2, 6),
                    ST_Multi(ST_MakePolygon(ST_MakeLine(ARRAY[
                        ST_MakePoint(lng0,        lat0),
                        ST_MakePoint(lng0 + side, lat0),
                        ST_MakePoint(lng0 + side, lat0 - side),
                        ST_MakePoint(lng0,        lat0 - side),
                        ST_MakePoint(lng0,        lat0)
                    ])))::geography,
                    'Active',
                    'Fallow'
                ) RETURNING id INTO block_id;

                -- The registered area is what PostGIS measures on the spheroid, rounded to the
                -- nearest tenth of a hectare the way a survey would report it.
                SELECT round((ST_Area(boundary) / 10000)::numeric, 1) INTO total_ha
                  FROM block WHERE id = block_id;

                -- Between 84% and 94% of a field is plantable; the rest is roads, drains and
                -- headlands, and varies block by block.
                plantable_ha := round(total_ha * (0.84 + ((block_no * 7) % 11) * 0.01), 1);

                UPDATE block
                   SET total_area_ha = total_ha,
                       plantable_area_ha = plantable_ha
                 WHERE id = block_id;

                unplantable := total_ha - plantable_ha;
                IF unplantable > 0 THEN
                    INSERT INTO block_non_plantable(block_id, reason_code, area_ha, remark) VALUES
                        (block_id, 'ROAD', round(unplantable * 0.45, 1), 'Haul road and headland'),
                        (block_id, 'CANAL', round(unplantable * 0.30, 1), 'Feeder canal');
                    IF block_no % 4 = 0 THEN
                        INSERT INTO block_non_plantable(block_id, reason_code, area_ha, remark)
                        VALUES (block_id, 'POND', round(unplantable * 0.10, 1), 'Farm reservoir');
                    ELSIF block_no % 5 = 0 THEN
                        INSERT INTO block_non_plantable(block_id, reason_code, area_ha, remark)
                        VALUES (block_id, 'FLOOD', round(unplantable * 0.10, 1), 'Seasonal flooding along the river');
                    END IF;
                END IF;
            END LOOP;
        END LOOP;
    END LOOP;

    -- Zone and farm boundaries are the union of what they contain, so the map draws one outline
    -- per level rather than a pile of overlapping block squares.
    UPDATE zone z SET boundary = sub.geom
      FROM (SELECT b.zone_id, ST_Multi(ST_Union(b.boundary::geometry))::geography AS geom
              FROM block b GROUP BY b.zone_id) sub
     WHERE sub.zone_id = z.id;

    UPDATE farm f SET boundary = sub.geom
      FROM (SELECT z.farm_id, ST_Multi(ST_Union(z.boundary::geometry))::geography AS geom
              FROM zone z WHERE z.boundary IS NOT NULL GROUP BY z.farm_id) sub
     WHERE sub.farm_id = f.id;
END
$seed$;

-- ---------------------------------------------------------------- planting records
--
-- Two crop years. 2025 is history and finished close to plan; 2026 is the year under management,
-- part recorded, so the variance and achievement figures have something real to show and the
-- monthly progress chart has a curve rather than a single bar.

DO $planting$
DECLARE
    r            record;
    season_2025  integer;
    season_2026  integer;
    varieties    integer[];
    idx          integer := 0;
    plan_new     numeric(12,4);
    plan_ratoon  numeric(12,4);
    variety      integer;
    plant_month  integer;
BEGIN
    SELECT id INTO season_2025 FROM crop_season WHERE code = 'CS2025';
    SELECT id INTO season_2026 FROM crop_season WHERE code = 'CS2026';
    SELECT array_agg(id ORDER BY id) INTO varieties FROM cane_variety;

    FOR r IN SELECT id, plantable_area_ha FROM block ORDER BY id LOOP
        idx := idx + 1;
        variety := varieties[1 + (idx % array_length(varieties, 1))];
        plant_month := 3 + (idx % 6);              -- March to August

        -- 2025: every third block was left fallow that year; the rest finished on or near plan.
        IF idx % 3 <> 0 THEN
            plan_new := round(r.plantable_area_ha * 0.55, 1);
            INSERT INTO block_planting(block_id, crop_season_id, crop_year, planting_year, planting_type,
                                       ratoon_no, cane_variety_id, planned_area_ha, actual_area_ha,
                                       planned_date, actual_date, expected_harvest_date)
            VALUES (r.id, season_2025, 2025, 2025, 'NewPlanting', 0, variety,
                    plan_new,
                    round(plan_new * (CASE WHEN idx % 4 = 0 THEN 0.92 ELSE 1.0 END), 1),
                    make_date(2025, plant_month, 8),
                    make_date(2025, plant_month, 8 + (idx % 10)),
                    make_date(2026, plant_month, 8));
        END IF;

        -- 2026: new planting on two blocks in three, ratoon where the 2025 crop is coming back.
        IF idx % 3 <> 1 THEN
            plan_new := round(r.plantable_area_ha * 0.40, 1);
            INSERT INTO block_planting(block_id, crop_season_id, crop_year, planting_year, planting_type,
                                       ratoon_no, cane_variety_id, planned_area_ha, actual_area_ha,
                                       planned_date, actual_date, expected_harvest_date)
            VALUES (r.id, season_2026, 2026, 2026, 'NewPlanting', 0, variety,
                    plan_new,
                    -- part of the programme is still to come, and two blocks in nine over-planted
                    CASE WHEN idx % 9 = 4 THEN round(plan_new * 1.08, 1)
                         WHEN idx % 5 = 0 THEN 0
                         ELSE round(plan_new * 0.85, 1) END,
                    make_date(2026, plant_month, 5),
                    CASE WHEN idx % 5 = 0 THEN NULL ELSE make_date(2026, plant_month, 5 + (idx % 12)) END,
                    make_date(2027, plant_month, 5));
        END IF;

        IF idx % 3 <> 2 THEN
            plan_ratoon := round(r.plantable_area_ha * 0.30, 1);
            INSERT INTO block_planting(block_id, crop_season_id, crop_year, planting_year, planting_type,
                                       ratoon_no, cane_variety_id, planned_area_ha, actual_area_ha,
                                       planned_date, actual_date, expected_harvest_date)
            VALUES (r.id, season_2026, 2026, 2026, 'Ratoon', 1, variety,
                    plan_ratoon,
                    round(plan_ratoon * (CASE WHEN idx % 7 = 0 THEN 0.6 ELSE 1.0 END), 1),
                    make_date(2026, 2 + (idx % 4), 12),
                    make_date(2026, 2 + (idx % 4), 12 + (idx % 8)),
                    make_date(2027, 2 + (idx % 4), 12));
        END IF;
    END LOOP;
END
$planting$;

-- A block carries cane once planting has been recorded against it for the current crop year.
UPDATE block b
   SET cane_status = 'Growing'
 WHERE EXISTS (SELECT 1 FROM block_planting bp
                WHERE bp.block_id = b.id AND bp.crop_year = 2026 AND bp.actual_area_ha > 0);

UPDATE block b
   SET cane_status = 'Prepared'
 WHERE b.cane_status = 'Fallow'
   AND EXISTS (SELECT 1 FROM block_planting bp
                WHERE bp.block_id = b.id AND bp.crop_year = 2026 AND bp.planned_area_ha > 0);

-- A couple of parcels the estate has taken out of production, so the land-status filter has
-- something to separate.
UPDATE block SET land_status = 'Reserved', remark = 'Held for the new weighbridge and workshop'
 WHERE code IN ('BLK-007');
UPDATE block SET land_status = 'Retired', cane_status = 'Fallow',
                 remark = 'Waterlogged since the 2024 floods; withdrawn from the planting programme'
 WHERE code IN ('BLK-018');

INSERT INTO schema_migrations(version) VALUES ('002_seed');
