-- append ID to non-root iteration and area paths
UPDATE iterations SET path=text2ltree(concat(path, concat('.',replace(cast(id as text), '-', '_')))) WHERE path!='' AND path IS NOT NULL;
UPDATE areas SET path=text2ltree(concat(path, concat('.',replace(cast(id as text), '-', '_')))) WHERE path!='' AND path IS NOT NULL;

-- update root iteration and area paths to use converted ids
UPDATE iterations SET path=text2ltree(replace(cast(id as text), '-', '_')) WHERE path='' OR PATH IS NULL;
UPDATE areas SET path=text2ltree(replace(cast(id as text), '-', '_')) WHERE path='' OR PATH IS NULL;

-- alter iteration and area path column to not accept NULL values
ALTER TABLE iterations ALTER COLUMN path SET NOT NULL;
ALTER TABLE areas ALTER COLUMN path SET NOT NULL;

-- drop constraints
ALTER TABLE iterations DROP CONSTRAINT iterations_name_space_id_path_unique;
ALTER TABLE areas DROP CONSTRAINT areas_name_space_id_path_unique;

-- TODO: (tkurian) this is broken -- add constraints for subpaths
-- CREATE UNIQUE INDEX areas_name_space_id_path_unique ON areas (space_id, subpath(path, 0, -1), name) WHERE deleted_at IS NULL;
-- CREATE UNIQUE INDEX iterations_name_space_id_path_unique ON iterations (space_id, subpath(path, 0, -1), name) WHERE deleted_at IS NULL;