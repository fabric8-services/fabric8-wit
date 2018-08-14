-- update root iteration and area paths to use converted ids
UPDATE iterations SET path=text2ltree(concat(path, concat('.',replace(cast(id as text), '-', '_')))) WHERE path!='' OR NOT NULL; 
UPDATE areas SET path=text2ltree(concat(path, concat('.',replace(cast(id as text), '-', '_')))) WHERE path!='' OR NOT NULL; 