-- Polarisation is a percentage and cannot exceed 100.
--
-- cane_sources has bounded its expected polarisation since migration 0013;
-- the deliveries measured against it did not, so a laboratory reading of
-- 140 % was accepted at the gate. The two tables disagreed about what a
-- valid reading is, and the looser one was the one recording real numbers.

ALTER TABLE daily_cane_supply
    ADD CONSTRAINT daily_cane_supply_pol_ck CHECK (pol_pct <= 100);
