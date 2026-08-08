-- Restore the stricter check. This only succeeds when no balance is negative,
-- which is the honest behaviour: reverting the rule cannot silently discard the
-- rows the rule was relaxed for.

ALTER TABLE stock_balances DROP CONSTRAINT stock_balances_hold_ck;
ALTER TABLE stock_balances ADD CONSTRAINT stock_balances_hold_ck
    CHECK (hold_quantity >= 0 AND hold_quantity <= quantity);
