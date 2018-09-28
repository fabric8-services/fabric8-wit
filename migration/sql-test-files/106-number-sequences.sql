SET sp_template.id = '{{index . 0}}';
SET sp1.id = '{{index . 1}}';
SET sp2.id = '{{index . 2}}';

SET wit.id = '{{index . 3}}';

SET wi1.id = '{{index . 4}}';
SET wi2.id = '{{index . 5}}';
SET wi3.id = '{{index . 6}}';
SET wi4.id = '{{index . 7}}';

SET iter1.id = '{{index . 8}}';
SET iter2.id = '{{index . 9}}';
SET iter3.id = '{{index . 10}}';
SET iter4.id = '{{index . 11}}';

SET area1.id = '{{index . 12}}';
SET area2.id = '{{index . 13}}';
SET area3.id = '{{index . 14}}';
SET area4.id = '{{index . 15}}';

-- create space template
INSERT INTO space_templates (id,name,description)
    VALUES(current_setting('sp_template.id')::uuid, current_setting('sp_template.id'), 'test template');

-- create two spaces
INSERT INTO spaces (id,name,space_template_id) 
    VALUES
        (current_setting('sp1.id')::uuid, current_setting('sp1.id'), current_setting('sp_template.id')::uuid),
        (current_setting('sp2.id')::uuid, current_setting('sp2.id'), current_setting('sp_template.id')::uuid);

INSERT INTO work_item_types (id,name,can_construct,space_template_id,fields,description,icon) 
    VALUES (current_setting('wit.id')::uuid, 'Custom WIT', 'true', current_setting('sp_template.id')::uuid, '{"system_title": {"type": {"kind": "string"}, "label": "Title", "required": true, "read_only": false, "description": "The title text of the work item"}}', 'Description for Impediment', 'fa fa-bookmark');

INSERT INTO work_items (id, type, space_id, fields, number, created_at)
VALUES
    (current_setting('wi1.id')::uuid, current_setting('wit.id')::uuid, current_setting('sp1.id')::uuid, '{"system_title":"WI1"}'::json, 1, '2018-09-17 16:01'),
    (current_setting('wi2.id')::uuid, current_setting('wit.id')::uuid, current_setting('sp1.id')::uuid, '{"system_title":"WI2"}'::json, 2, '2018-09-17 18:01'),
    (current_setting('wi3.id')::uuid, current_setting('wit.id')::uuid, current_setting('sp2.id')::uuid, '{"system_title":"WI3"}'::json, 1, '2018-09-17 12:01'),
    (current_setting('wi4.id')::uuid, current_setting('wit.id')::uuid, current_setting('sp2.id')::uuid, '{"system_title":"WI4"}'::json, 2, '2018-09-17 17:01');

INSERT INTO work_item_number_sequences (space_id, current_val)
VALUES 
    (current_setting('sp1.id')::uuid, 2),
    (current_setting('sp2.id')::uuid, 2);

INSERT INTO iterations (id, name, path, space_id, created_at) 
VALUES 
    (current_setting('iter1.id')::uuid, 'iteration 1', replace(current_setting('iter1.id'), '-', '_')::ltree, current_setting('sp1.id')::uuid, '2018-09-17 16:01'),
    (current_setting('iter2.id')::uuid, 'iteration 2', replace(current_setting('iter2.id'), '-', '_')::ltree, current_setting('sp1.id')::uuid, '2018-09-17 15:01'),
    (current_setting('iter3.id')::uuid, 'iteration 3', replace(current_setting('iter3.id'), '-', '_')::ltree, current_setting('sp2.id')::uuid, '2018-09-17 16:01'),
    (current_setting('iter4.id')::uuid, 'iteration 4', replace(current_setting('iter4.id'), '-', '_')::ltree, current_setting('sp2.id')::uuid, '2018-09-17 15:01');

INSERT INTO areas (id, name, path, space_id, created_at) 
VALUES 
    (current_setting('area1.id')::uuid, 'area 1', replace(current_setting('area1.id'), '-', '_')::ltree, current_setting('sp1.id')::uuid, '2018-09-17 13:01'),
    (current_setting('area2.id')::uuid, 'area 2', replace(current_setting('area2.id'), '-', '_')::ltree, current_setting('sp1.id')::uuid, '2018-09-17 12:01'),
    (current_setting('area3.id')::uuid, 'area 3', replace(current_setting('area3.id'), '-', '_')::ltree, current_setting('sp2.id')::uuid, '2018-09-17 13:01'),
    (current_setting('area4.id')::uuid, 'area 4', replace(current_setting('area4.id'), '-', '_')::ltree, current_setting('sp2.id')::uuid, '2018-09-17 12:01');
