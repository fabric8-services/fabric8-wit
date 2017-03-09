CREATE EXTENSION IF NOT EXISTS "ltree";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

------------------------------------------------------------------------------
-- Update the work item type table itself:
--
-- 0. In parallel to the current primary key ("name" column), we'll add a column
-- "id" that will become the new primary key later down the road.
--
-- 1. For all system-defined WITs, do not use the UUID as it was generated
-- during the migration instead use the one that is defined in the code.
--
-- 2. Add new "description" column and fill with the default value of 'This is
-- the description for the work item type "X".'.
--
-- 3. Update the "path" column of the WIT table to use the new UUID (with "-"
-- replaced by "_") instead of the "name" column.
--
-- 4. Drop the constraint that limits the "name" column to be contain only
-- C-LOCALE characters. This is a human readable free form field now.
--
-- 5. Finally, switch to "id" column to be our new primary key.
-------------------------------------------------------------------------------

ALTER TABLE work_item_types ADD COLUMN id uuid DEFAULT uuid_generate_v4() NOT NULL;

-- Use WIT IDs define in the code
UPDATE work_item_types SET id = '86af5178-9b41-469b-9096-57e5155c3f31' WHERE name = 'planneritem';
UPDATE work_item_types SET id = 'bbf35418-04b6-426c-a60b-7f80beb0b624' WHERE name = 'userstory';
UPDATE work_item_types SET id = '3194ab60-855b-4155-9005-9dce4a05f1eb' WHERE name = 'valueproposition';
UPDATE work_item_types SET id = 'ee7ca005-f81d-4eea-9b9b-1965df0988d0' WHERE name = 'fundamental';
UPDATE work_item_types SET id = 'b9a71831-c803-4f66-8774-4193fffd1311' WHERE name = 'experience';
UPDATE work_item_types SET id = '0a24d3c2-e0a6-4686-8051-ec0ea1915a28' WHERE name = 'feature';
UPDATE work_item_types SET id = '71171e90-6d35-498f-a6a7-2083b5267c18' WHERE name = 'scenario';
UPDATE work_item_types SET id = '26787039-b68f-4e28-8814-c2f93be1ef4e' WHERE name = 'bug';

ALTER TABLE work_item_types ADD COLUMN description text;
UPDATE work_item_types SET description = concat('This is the description for the work item type "', name, '".');

CREATE OR REPLACE FUNCTION UUIDToLtreeNode(u uuid, OUT node ltree) AS $$ BEGIN
-- Converts a UUID value into a value usable inside an Ltree 
    SELECT replace(u::text, '-', '_') INTO node;
END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION LtreeNodeToUUID(node ltree, OUT u uuid) AS $$ BEGIN
-- Converts an Ltree node into a UUID value 
    SELECT replace(node::text, '_', '-') INTO u;
END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_new_wit_path(oldPath ltree, OUT newPath ltree) AS $$
-- Converts the oldPath ltree value which was planneritem.bug and so forth into
-- an ltree that is based on the UUID of a work item type.
    DECLARE
        nodeName text;
        nodeId text;
        newPathArray text array;
    BEGIN
        FOREACH nodeName IN array regexp_split_to_array(oldPath::text,'\.')
        LOOP
            SELECT UUIDToLtreeNode(id) INTO nodeId FROM work_item_types WHERE name = nodeName;
            newPathArray := array_append(newPathArray, nodeId);
        END LOOP;
        newPath := array_to_string(newPathArray, '.');
    END;
$$ LANGUAGE plpgsql;

UPDATE work_item_types SET path = get_new_wit_path(path);

DROP FUNCTION get_new_wit_path(oldPath ltree, OUT newPath ltree);

-- Drop constraints that depend on the primary key.
ALTER TABLE work_item_link_types DROP CONSTRAINT work_item_link_types_source_type_name_fkey;
ALTER TABLE work_item_link_types DROP CONSTRAINT work_item_link_types_target_type_name_fkey;
-- Drop the primary key itself and set up the new one on the "id" column.
ALTER TABLE work_item_types DROP CONSTRAINT work_item_types_pkey;
ALTER TABLE work_item_types ADD PRIMARY KEY (id);
ALTER TABLE work_item_types DROP CONSTRAINT work_item_link_types_check_name_c_locale;

------------------------------------------------------------------------------
-- Update all references to the work item type table to point to the new "id"
-- column instead of the "name" column. Since this involves column type change
-- from "text" to "uuid" we'll simply add a new reference and delete the old
-- one.
------------------------------------------------------------------------------

------------------------------
-- Update work item link types
------------------------------

ALTER TABLE work_item_link_types ADD COLUMN source_type_id uuid REFERENCES work_item_types(id) ON DELETE CASCADE;
ALTER TABLE work_item_link_types ADD COLUMN target_type_id uuid REFERENCES work_item_types(id) ON DELETE CASCADE;

UPDATE work_item_link_types SET source_type_id = (SELECT id FROM work_item_types WHERE name = source_type_name);
UPDATE work_item_link_types SET target_type_id = (SELECT id FROM work_item_types WHERE name = target_type_name);

ALTER TABLE work_item_link_types DROP COLUMN source_type_name;
ALTER TABLE work_item_link_types DROP COLUMN target_type_name;

-- Add NOT NULL criteria 
ALTER TABLE work_item_link_types ALTER COLUMN source_type_id SET NOT NULL;
ALTER TABLE work_item_link_types ALTER COLUMN target_type_id SET NOT NULL;

--------------------
-- Update work items
--------------------

-- NOTE: The foreign key is new!
ALTER TABLE work_items RENAME type TO type_old;
ALTER TABLE work_items ADD COLUMN type uuid REFERENCES work_item_types(id) ON DELETE CASCADE;
UPDATE work_items SET type = (SELECT id FROM work_item_types WHERE name = type_old);
ALTER TABLE work_items DROP COLUMN type_old;

-- Add NOT NULL criteria
ALTER TABLE work_items ALTER COLUMN type SET NOT NULL;