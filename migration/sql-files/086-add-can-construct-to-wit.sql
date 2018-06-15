CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

SET LOCAL idx.planner_item_type_id = '{{index . 0}}';

-- Add boolean can_construct field to work item type table and make it default
-- to TRUE.
ALTER TABLE work_item_types ADD COLUMN can_construct boolean;
UPDATE work_item_types SET can_construct = TRUE;
UPDATE work_item_types SET can_construct = FALSE WHERE id = current_setting('idx.planner_item_type_id')::uuid;
ALTER TABLE work_item_types ALTER can_construct SET DEFAULT TRUE;
ALTER TABLE work_item_types ALTER COLUMN can_construct SET NOT NULL;