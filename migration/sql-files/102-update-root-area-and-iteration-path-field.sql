UPDATE iterations SET path=text2ltree(replace(cast(id as text), '-', '_')) WHERE path='' OR PATH IS NULL;
UPDATE areas SET path=text2ltree(replace(cast(id as text), '-', '_')) WHERE path='' OR PATH IS NULL;

ALTER TABLE iterations ALTER COLUMN path SET NOT NULL;
ALTER TABLE areas ALTER COLUMN path SET NOT NULL;