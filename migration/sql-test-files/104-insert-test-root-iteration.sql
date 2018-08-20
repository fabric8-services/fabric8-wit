SET sp_template.id = '0961ada9-afca-439f-865b-d5a8c6e3e9a2';
SET sp.id = '{{index . 0}}';
SET iter.root_empty_path = '{{index . 1}}';
SET iter.root_null_path = '{{index . 2}}';
SET iter.child_of_empty_path = '{{index . 3}}';
SET iter.child_of_null_path = '{{index . 4}}';

SET area.root_empty_path = '{{index . 5}}';
SET area.root_null_path = '{{index . 6}}';
SET area.child_of_empty_path = '{{index . 7}}';
SET area.child_of_null_path = '{{index . 8}}';

-- create space template
INSERT INTO space_templates (id,name,description)
    VALUES(current_setting('sp_template.id')::uuid, 'test space template 2', 'test template');

-- create space
INSERT INTO spaces (id,name,space_template_id) 
    VALUES (current_setting('sp.id')::uuid, 'test space 2', current_setting('sp_template.id')::uuid);

-- create iterations:
--
--    with_empty_path
--    |
--    |___ child_of_empty_path
--
--    with_null_path
--    |
--    |___ child_of_null_path
INSERT INTO iterations (id, name, path, space_id) 
    VALUES 
        (current_setting('iter.root_empty_path')::uuid, 'with_empty_path', '', current_setting('sp.id')::uuid),
        (current_setting('iter.root_null_path')::uuid, 'with_null_path', NULL, current_setting('sp.id')::uuid),
        (current_setting('iter.child_of_empty_path')::uuid, 'child_of_empty_path', replace(current_setting('iter.root_empty_path'), '-', '_')::ltree, current_setting('sp.id')::uuid),
        (current_setting('iter.child_of_null_path')::uuid, 'child_of_null_path', replace(current_setting('iter.root_null_path'), '-', '_')::ltree, current_setting('sp.id')::uuid);

-- create areas:
--
--    with_empty_path
--    |
--    |___ child_of_empty_path
--
--    with_null_path
--    |
--    |___ child_of_null_path
INSERT INTO areas (id, name, path, space_id)
    VALUES
        (current_setting('area.root_empty_path')::uuid, 'root_empty_path', '', current_setting('sp.id')::uuid),
        (current_setting('area.root_null_path')::uuid, 'root_null_path', NULL, current_setting('sp.id')::uuid),
        (current_setting('area.child_of_empty_path')::uuid, 'child_of_empty_path', replace(current_setting('area.root_empty_path'), '-', '_')::ltree, current_setting('sp.id')::uuid),
        (current_setting('area.child_of_null_path')::uuid, 'child_of_empty_path', replace(current_setting('area.root_null_path'), '-', '_')::ltree, current_setting('sp.id')::uuid);