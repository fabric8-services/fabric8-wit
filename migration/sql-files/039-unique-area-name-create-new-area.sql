------ You can't allow the same area name and the same ancestry inside a space

ALTER TABLE areas ADD CONSTRAINT areas_name_space_id_path_unique UNIQUE(space_id,name,path);

------  For existing spaces in production, which dont have a default area, create one.
--
-- 1. Get all spaces which have an area under it with the same name.
-- 2. Get all spaces not in (1)
-- 3. insert an 'area' for all such spaces in (2)

INSERT INTO areas
            (created_at,
             updated_at,
             name,
             space_id)
SELECT current_timestamp,
       current_timestamp,
       name,
       id
FROM   spaces
WHERE  id NOT IN (SELECT s.id
                  FROM   spaces AS s
                         INNER JOIN areas AS a
                                 ON s.name = a.name
                                    AND s.id = a.space_id);  