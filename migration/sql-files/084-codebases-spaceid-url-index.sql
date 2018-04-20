-- Delete duplicate entries of codebases in same space
-- See here: https://wiki.postgresql.org/wiki/Deleting_duplicates
DELETE FROM codebases
WHERE id IN (
    SELECT id
    FROM (
        SELECT id, ROW_NUMBER() OVER (partition BY url, space_id ORDER BY deleted_at DESC) AS rnum
        FROM codebases
    ) t
    WHERE t.rnum > 1
);

-- From now on ensure we have codebase only once in space
CREATE UNIQUE INDEX codebases_spaceid_url_idx
ON codebases (url, space_id) WHERE deleted_at IS NULL;
