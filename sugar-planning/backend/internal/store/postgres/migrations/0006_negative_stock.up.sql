-- 0006_negative_stock: let an authorised posting drive a balance below zero.
--
-- The original check on stock_balances was "hold_quantity >= 0 AND
-- hold_quantity <= quantity". With a hold of zero that reads as "0 <= quantity",
-- which forbids a negative balance outright - and that contradicts the posting
-- rules, where a negative balance is refused by default but permitted for a
-- caller holding the override (domain.PostingOptions.AllowNegativeStock). Sites
-- that book consumption before the matching receipt need it, and a reversal of
-- a receipt whose stock has since moved on needs it too.
--
-- The rule the constraint should express is the one about holds, not the one
-- about negative stock: you cannot hold more sugar than is physically there,
-- and when the balance is negative there is nothing to hold at all. Whether a
-- negative balance is allowed stays a business decision, checked in the domain
-- against the caller's permission and recorded on the audit event.

ALTER TABLE stock_balances DROP CONSTRAINT stock_balances_hold_ck;
ALTER TABLE stock_balances ADD CONSTRAINT stock_balances_hold_ck
    CHECK (hold_quantity >= 0 AND hold_quantity <= GREATEST(quantity, 0));
