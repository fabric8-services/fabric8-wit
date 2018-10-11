SET sp_template.id = '{{index . 0}}';
SET sp1.id = '{{index . 1}}';
SET sp2.id = '{{index . 2}}';

SET iter1.id = '{{index . 3}}';
SET iter2.id = '{{index . 4}}';
SET iter3.id = '{{index . 5}}';
SET iter4.id = '{{index . 6}}';

-- create space template
INSERT INTO space_templates (id,name,description)
    VALUES(current_setting('sp_template.id')::uuid, current_setting('sp_template.id'), 'test template');

-- create two spaces
INSERT INTO spaces (id,name,space_template_id) 
    VALUES
        (current_setting('sp1.id')::uuid, current_setting('sp1.id'), current_setting('sp_template.id')::uuid),
        (current_setting('sp2.id')::uuid, current_setting('sp2.id'), current_setting('sp_template.id')::uuid);

INSERT INTO iterations (id, name, path, space_id, created_at) 
VALUES 
    (current_setting('iter1.id')::uuid, 'iteration 1', replace(current_setting('iter1.id'), '-', '_')::ltree, current_setting('sp1.id')::uuid, '2018-09-17 16:01'),
    (current_setting('iter2.id')::uuid, 'iteration 2', replace(current_setting('iter2.id'), '-', '_')::ltree, current_setting('sp1.id')::uuid, '2018-09-17 15:01'),
    (current_setting('iter3.id')::uuid, 'iteration 3', replace(current_setting('iter3.id'), '-', '_')::ltree, current_setting('sp2.id')::uuid, '2018-09-17 16:01'),
    (current_setting('iter4.id')::uuid, 'iteration 4', replace(current_setting('iter4.id'), '-', '_')::ltree, current_setting('sp2.id')::uuid, '2018-09-17 15:01');
