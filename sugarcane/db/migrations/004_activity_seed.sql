-- The nineteen sample planting activities of section 6, with their standards, and the dependency
-- chain that runs through them.
--
-- The offsets are days from the projection's planting start: land is surveyed forty-five days
-- before the cane goes in, and the last herbicide goes on sixty-five days after.

INSERT INTO planting_activity(
    company_id, code, name, category, applicable_crop_type, sequence_no, standard_start_day_offset,
    standard_capacity_per_hour, standard_capacity_per_day, standard_duration_per_ha,
    standard_labour_days_per_ha, required_equipment_category,
    is_mandatory, requires_tractor, requires_equipment, requires_material, requires_labour, allow_overlap)
SELECT c.id, v.code, v.name, v.category::activity_category, v.crop::applicable_crop_type, v.seq, v.offset_days,
       v.per_hour, v.per_day, v.hours_per_ha, v.labour_days_per_ha, v.equipment_category,
       v.mandatory, v.tractor, v.equipment, v.material, v.labour, v.overlap
  FROM company c,
  (VALUES
    ('A001','Land survey','Survey','NewPlanting',1,-45,3.0,20.0,0.4,0.2,NULL,true,false,false,false,true,true),
    ('A002','Land clearing','LandPreparation','NewPlanting',2,-40,0.5,4.0,2.0,1.2,'Other',true,true,true,false,true,false),
    ('A003','First plowing','LandPreparation','NewPlanting',3,-35,0.6,4.5,1.8,0.3,'DiscPlow',true,true,true,false,false,false),
    ('A004','Second plowing','LandPreparation','NewPlanting',4,-30,0.6,4.5,1.8,0.3,'MoldboardPlow',true,true,true,false,false,false),
    ('A005','Harrowing','LandPreparation','Both',5,-25,0.9,7.0,1.2,0.2,'Harrow',true,true,true,false,false,false),
    ('A006','Land leveling','LandPreparation','NewPlanting',6,-20,0.5,4.0,2.0,0.3,'LandLeveler',true,true,true,false,false,false),
    ('A007','Drainage preparation','LandPreparation','NewPlanting',7,-16,0.4,3.0,2.6,0.8,'Other',false,true,true,false,true,false),
    ('A008','Furrow preparation','LandPreparation','NewPlanting',8,-12,0.8,6.0,1.4,0.2,'FurrowOpener',true,true,true,false,false,false),
    ('A009','Seed cane cutting','SeedPreparation','NewPlanting',9,-8,1.0,8.0,1.0,1.5,'SeedCutter',true,false,true,false,true,false),
    ('A010','Seed cane transportation','SeedPreparation','NewPlanting',10,-5,1.5,12.0,0.7,0.6,'Trailer',true,true,true,false,true,false),
    ('A011','Seed treatment','SeedPreparation','NewPlanting',11,-3,1.2,9.0,0.9,0.5,NULL,true,false,false,true,true,false),
    ('A012','Planting','Planting','Both',12,0,0.5,4.0,2.2,2.5,'CanePlanter',true,true,true,true,true,false),
    ('A013','Basal fertilizer application','Fertilization','Both',13,1,1.2,9.0,0.9,0.4,'FertilizerSpreader',true,true,true,true,true,false),
    ('A014','Pre-emergence herbicide application','WeedControl','Both',14,3,1.5,11.0,0.8,0.3,'ChemicalSprayer',true,true,true,true,true,false),
    ('A015','Irrigation','Irrigation','Both',15,5,1.0,8.0,1.0,0.5,'WaterTruck',true,true,true,true,true,false),
    ('A016','Gap filling','CropCare','Both',16,30,0.8,6.0,1.3,1.8,NULL,false,false,false,true,true,true),
    ('A017','First cultivation','CropCare','Both',17,45,0.9,7.0,1.2,0.3,'Harrow',true,true,true,false,false,false),
    ('A018','First fertilizer application','Fertilization','Both',18,55,1.2,9.0,0.9,0.4,'FertilizerSpreader',true,true,true,true,true,false),
    ('A019','Post-emergence herbicide application','WeedControl','Both',19,65,1.5,11.0,0.8,0.3,'ChemicalSprayer',true,true,true,true,true,false)
  ) AS v(code, name, category, crop, seq, offset_days, per_hour, per_day, hours_per_ha,
         labour_days_per_ha, equipment_category, mandatory, tractor, equipment, material, labour, overlap)
 WHERE c.code = 'SGC';

-- Each activity waits for the one before it, which is how field work runs. Planting waits a day
-- after the furrows are opened; the two optional activities — drainage preparation and gap filling
-- — only warn, so a plan can proceed past them under pressure.
INSERT INTO activity_dependency(activity_id, depends_on_id, lag_days, is_blocking)
SELECT a.id, prev.id,
       CASE WHEN a.code = 'A012' THEN 1 ELSE 0 END,
       a.is_mandatory
  FROM planting_activity a
  JOIN planting_activity prev
    ON prev.company_id = a.company_id AND prev.sequence_no = a.sequence_no - 1
 WHERE a.sequence_no > 1;

INSERT INTO schema_migrations(version) VALUES ('004_activity_seed');
