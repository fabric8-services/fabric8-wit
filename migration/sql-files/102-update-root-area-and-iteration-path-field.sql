-- update root iteration and area paths to use converted ids
UPDATE iterations SET path=text2ltree(replace(cast(id as text), '-', '_')) WHERE path='' OR PATH IS NULL;
UPDATE areas SET path=text2ltree(replace(cast(id as text), '-', '_')) WHERE path='' OR PATH IS NULL;

-- alter iteration and area path column to not accept NULL values
ALTER TABLE iterations ALTER COLUMN path SET NOT NULL;
ALTER TABLE areas ALTER COLUMN path SET NOT NULL;