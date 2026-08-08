-- Which worksheet the plan is on.
--
-- The importer read the first sheet of a workbook and nothing else, which is
-- fine for a file exported for the purpose and wrong for a real one. The mill's
-- own production plan opens on a summary tab and keeps the three hundred days
-- of daily figures on the second; asked to read it, the importer returned
-- twenty-three rows of the summary interpreted as dates and tonnages, with no
-- indication that it had read the wrong tab at all.

ALTER TABLE import_mappings ADD COLUMN sheet text NOT NULL DEFAULT '';
