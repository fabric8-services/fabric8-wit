-- See https://www.postgresql.org/docs/current/static/ltree.html for the
-- reference See http://leapfrogonline.io/articles/2015-05-21-postgres-ltree/
-- for an explanation
CREATE EXTENSION IF NOT EXISTS "ltree";

-- The following update needs to be done in order to get the WIT storage in a
-- good shape for it to be migrated to an ltree
UPDATE work_item_types SET
    -- Remove any leading '/' from the WIT's path.
    -- Remove any occurence of 'system.'.
    -- Replace '/' with '.' as the new path separator for use with ltree.
    -- Replace every non-C-LOCALE character with an underscore (the "." is an
    -- exception because it will be used by ltree)
    path =  regexp_replace(
                replace(replace(ltrim(path, '/'), 'system.', ''), '/', '.'),
                '[^a-zA-Z0-9_\.]',
                '_'
            )
    ;

-- Convert the path column from type text to ltree
ALTER TABLE work_item_types ALTER COLUMN path TYPE ltree USING path::ltree;

-- Use the leaf of the path "tree" as the name of the work item type
UPDATE work_item_types SET name = subpath(path,-1,1);

-- Add a constraint to the work item type name 
ALTER TABLE work_item_types ADD CONSTRAINT work_item_link_types_check_name_c_locale CHECK (name ~ '[a-zA-Z0-9_]');

-- Add indexes 
CREATE INDEX wit_path_gist_idx ON work_item_types USING GIST (path);
CREATE INDEX wit_path_idx ON work_item_types USING BTREE (path);
