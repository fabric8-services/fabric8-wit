CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

SET LOCAL idx.legacy_space_template_id = '{{index . 0}}';
SET LOCAL idx.base_space_template_id = '{{index . 1}}';
SET LOCAL idx.planner_item_type_id = '{{index . 2}}';

-- create base space template
INSERT INTO space_templates (id, name, description) VALUES(
    current_setting('idx.base_space_template_id')::uuid,
    'base space template',
    'this will be overwritten by the base space template when common types are populated'
);

-- Add foreign key to work item type relation and make all but the planner item
-- type a part of the base template.
UPDATE work_item_types SET space_template_id = current_setting('idx.base_space_template_id')::uuid WHERE id = current_setting('idx.planner_item_type_id')::uuid;

UPDATE work_item_types SET space_template_id = current_setting('idx.legacy_space_template_id')::uuid WHERE id <> current_setting('idx.planner_item_type_id')::uuid;

-- Add foreign key to work item link type relation and make all existing link
-- types a part of the base template.
UPDATE work_item_link_types SET space_template_id = current_setting('idx.base_space_template_id')::uuid;
