-- drop existing unique constraint
ALTER table labels DROP CONSTRAINT labels_name_space_id_unique;
-- create unique index on two columns
CREATE UNIQUE INDEX labels_name_space_id_unique_idx ON labels (space_id, name) WHERE deleted_at IS NULL;
