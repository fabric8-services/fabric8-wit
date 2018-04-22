CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE type_group_bucket_enum AS ENUM('portfolio', 'requirement', 'iteration');

CREATE TABLE work_item_type_groups (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    version integer DEFAULT 0 NOT NULL,
    position integer DEFAULT 0 NOT NULL,
    name text NOT NULL CHECK(name <> ''),
    bucket type_group_bucket_enum NOT NULL,
    icon text,
    space_template_id uuid REFERENCES space_templates(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX work_item_type_groups_name_uidx ON work_item_type_groups (name, space_template_id) WHERE deleted_at IS NULL;

CREATE TABLE work_item_type_group_members (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    type_group_id uuid REFERENCES work_item_type_groups(id) ON DELETE CASCADE,
    work_item_type_id uuid REFERENCES work_item_types(id) ON DELETE CASCADE,
    position integer DEFAULT 0 NOT NULL
);

CREATE UNIQUE INDEX work_item_type_group_members_uidx ON work_item_type_group_members (type_group_id, work_item_type_id) WHERE deleted_at IS NULL;

CREATE TABLE work_item_child_types (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_work_item_type_id uuid REFERENCES work_item_types(id) ON DELETE CASCADE,
    child_work_item_type_id uuid REFERENCES work_item_types(id) ON DELETE CASCADE,
    position integer DEFAULT 0 NOT NULL
);

CREATE UNIQUE INDEX work_item_child_types_uidx ON work_item_child_types (parent_work_item_type_id, child_work_item_type_id) WHERE deleted_at IS NULL;

-- Only allow one work item link type with the same name for the same space
-- template in existence.
CREATE UNIQUE INDEX work_item_link_types_name_idx ON work_item_link_types (name, space_template_id) WHERE deleted_at IS NULL;