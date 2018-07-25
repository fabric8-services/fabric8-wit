-- Add action rules fields to work_item_types. The values are
-- optional, so they may be null and are being kept at null
-- on existing WITs.
ALTER TABLE work_item_types ADD COLUMN trans_rule_key text;
ALTER TABLE work_item_types ADD COLUMN trans_rule_argument text;
