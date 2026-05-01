ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS aff_frozen_quota DECIMAL(20,8) NOT NULL DEFAULT 0;

COMMENT ON COLUMN user_affiliates.aff_frozen_quota IS 'Rebate quota currently frozen (pending thaw after freeze period)';

ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS frozen_until TIMESTAMPTZ NULL;

COMMENT ON COLUMN user_affiliate_ledger.frozen_until IS 'Rebate frozen until this time; NULL means already thawed or never frozen';

CREATE INDEX IF NOT EXISTS idx_ual_frozen_thaw
    ON user_affiliate_ledger (user_id, frozen_until)
    WHERE frozen_until IS NOT NULL;
