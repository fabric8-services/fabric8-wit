CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

SET LOCAL idx.space_template_id = '{{index . 0}}';
SET LOCAL idx.planner_item_type_id = '{{index . 1}}';

-- Remove space_id field from link types (WILTs) and work item types (WITs) This
-- can be done because all WILTs and WITs exist in the system space anyway. So
-- in order to maintain a compatibility with the current API on controller level
-- we can just fake the space relationship to be pointing to the system space.
ALTER TABLE work_item_link_types DROP COLUMN space_id;
ALTER TABLE work_item_types DROP COLUMN space_id;

CREATE TABLE space_templates (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    version integer DEFAULT 0 NOT NULL,
    name text NOT NULL CHECK(name <> ''),
    description text,
    can_construct boolean DEFAULT TRUE NOT NULL
);
CREATE UNIQUE INDEX space_templates_name_uidx ON space_templates (name) WHERE deleted_at IS NULL;

-- Create a default empty space template
INSERT INTO space_templates (id, name, description) VALUES(
    current_setting('idx.space_template_id')::uuid,
    'empty space template',
    'this will be overwritten by the legacy space template when common types are populated'
);

-- Add foreign key to spaces relation
ALTER TABLE spaces ADD COLUMN space_template_id uuid REFERENCES space_templates(id) ON DELETE CASCADE;
UPDATE spaces SET space_template_id = current_setting('idx.space_template_id')::uuid;
ALTER TABLE spaces ALTER COLUMN space_template_id SET NOT NULL;

-- Add foreign key to work item type relation
ALTER TABLE work_item_types ADD COLUMN space_template_id uuid REFERENCES space_templates(id) ON DELETE CASCADE;
UPDATE work_item_types SET space_template_id = current_setting('idx.space_template_id')::uuid;
ALTER TABLE work_item_types ALTER COLUMN space_template_id SET NOT NULL;

-- Add foreign key to work item link type relation
ALTER TABLE work_item_link_types ADD COLUMN space_template_id uuid REFERENCES space_templates(id) ON DELETE CASCADE;
UPDATE work_item_link_types SET space_template_id = current_setting('idx.space_template_id')::uuid;
ALTER TABLE work_item_link_types ALTER COLUMN space_template_id SET NOT NULL;
