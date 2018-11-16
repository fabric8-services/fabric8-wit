SET sp_template.id = '{{index . 0}}';

SET link_category.id = '{{index . 1}}';

SET link_type1.id = '{{index . 2}}';
SET link_type2.id = '{{index . 3}}';

-- create space template
INSERT INTO space_templates (id,name,description)
    VALUES (current_setting('sp_template.id')::uuid, current_setting('sp_template.id'), 'test template');

-- create a link category
INSERT INTO work_item_link_categories (id, name)
    VALUES (current_setting('link_category.id')::uuid, current_setting('link_category.id'));

-- create a link type, that references a link category
INSERT INTO work_item_link_types (id,space_template_id,name,forward_name,reverse_name,topology,link_category_id) 
    VALUES (current_setting('link_type1.id')::uuid, current_setting('sp_template.id')::uuid, current_setting('link_type1.id')::uuid, 'forward', 'reverse', 'network', current_setting('link_category.id')::uuid);

-- create a link type, that doesn't reference a link category
INSERT INTO work_item_link_types (id,space_template_id,name,forward_name,reverse_name,topology) 
    VALUES
        (current_setting('link_type2.id')::uuid, current_setting('sp_template.id')::uuid, current_setting('link_type2.id')::uuid, 'forward', 'reverse', 'network');