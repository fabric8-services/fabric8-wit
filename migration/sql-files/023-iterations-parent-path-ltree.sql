CREATE EXTENSION IF NOT EXISTS "ltree";

-- Rename parent_id column
ALTER TABLE iterations RENAME parent_id to parent_path;

-- Need to convert the parent_path column to text in order to
-- replace non-locale characters with an underscore
ALTER TABLE iterations ALTER parent_path TYPE text USING parent_path::text;

-- Need to update values of Iteration's' ParentID in order to migrate it to ltree
-- Replace every non-C-LOCALE character with an underscore
UPDATE iterations SET parent_path = regexp_replace(parent_path, '[^a-zA-Z0-9_\.]', '_', 'g');

-- Finally values in parent_path are now in good shape for ltree and can be casted automatically to type ltree
-- Convert the parent column from type uuid to ltree
ALTER TABLE iterations ALTER parent_path TYPE ltree USING parent_path::ltree;

-- Enable full text search operaions using GIST index on parent_path
CREATE INDEX iteration_parent_path_gist_idx ON iterations USING GIST (parent_path);

-- Enable equality and range operations < <= = >= > using BTREE index on parent_path
CREATE INDEX iteration_parent_path_idx ON iterations USING BTREE (parent_path);
