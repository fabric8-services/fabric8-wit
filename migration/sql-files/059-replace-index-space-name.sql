-- drop existing unique index
DROP INDEX spaces_name_idx;
-- recreate unique index with original index and lowercase name, on two columns
CREATE UNIQUE INDEX spaces_name_idx ON spaces (lower(name), owner_id) WHERE deleted_at IS NULL;
