-- update root iteration and area paths to use converted ids
UPDATE iterations SET path=text2ltree(concat(path, concat('.',replace(cast(id as text), '-', '_')))) WHERE path!='' OR NOT NULL; 
UPDATE areas SET path=text2ltree(concat(path, concat('.',replace(cast(id as text), '-', '_')))) WHERE path!='' OR NOT NULL; 

ALTER TABLE iterations DROP CONSTRAINT iterations_name_space_id_path_unique;
ALTER TABLE areas DROP CONSTRAINT areas_name_space_id_path_unique;

CREATE UNIQUE INDEX areas_name_space_id_path_unique ON areas (space_id, subpath(path, 0, -1), name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX iterations_name_space_id_path_unique ON iterations (space_id, subpath(path, 0, -1), name) WHERE deleted_at IS NULL;