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
    VALUES (current_setting('wit.id')::uuid, 'Impediment', 'true', current_setting('sp_template.id')::uuid, '{"resolution": {"type": {"values": ["Done", "Rejected", "Duplicate", "Incomplete Description", "Can not Reproduce", "Partially Completed", "Deferred", "Wont Fix", "Out of Date", "Explained", "Verified"], "base_type": {"kind": "string"}, "simple_type": {"kind": "enum"}, "rewritable_values": false}, "label": "Resolution", "required": false, "read_only": false, "description": "The reason why this work items state was last changed.\n"}, "system.area": {"type": {"kind": "area"}, "label": "Area", "required": false, "read_only": false, "description": "The area to which the work item belongs"}, "system.order": {"type": {"kind": "float"}, "label": "Execution Order", "required": false, "read_only": true, "description": "Execution Order of the workitem"}, "system.state": {"type": {"values": ["New", "Open", "In Progress", "Resolved", "Closed"], "base_type": {"kind": "string"}, "simple_type": {"kind": "enum"}, "rewritable_values": false}, "label": "State", "required": true, "read_only": false, "description": "The state of the impediment."}, "system.title": {"type": {"kind": "string"}, "label": "Title", "required": true, "read_only": false, "description": "The title text of the work item"}, "system.labels": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "label"}}, "label": "Labels", "required": false, "read_only": false, "description": "List of labels attached to the work item"}, "system.number": {"type": {"kind": "integer"}, "label": "Number", "required": false, "read_only": true, "description": "The unique number that was given to this workitem within its space."}, "system.creator": {"type": {"kind": "user"}, "label": "Creator", "required": true, "read_only": false, "description": "The user that created the work item"}, "system.codebase": {"type": {"kind": "codebase"}, "label": "Codebase", "required": false, "read_only": false, "description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "user"}}, "label": "Assignees", "required": false, "read_only": false, "description": "The users that are assigned to the work item"}, "system.iteration": {"type": {"kind": "iteration"}, "label": "Iteration", "required": false, "read_only": false, "description": "The iteration to which the work item belongs"}, "system.created_at": {"type": {"kind": "instant"}, "label": "Created at", "required": false, "read_only": true, "description": "The date and time when the work item was created"}, "system.updated_at": {"type": {"kind": "instant"}, "label": "Updated at", "required": false, "read_only": true, "description": "The date and time when the work item was last updated"}, "system.description": {"type": {"kind": "markup"}, "label": "Description", "required": false, "read_only": false, "description": "A descriptive text of the work item"}, "system.remote_item_id": {"type": {"kind": "string"}, "label": "Remote item", "required": false, "read_only": false, "description": "The ID of the remote work item"}}', 'Description for Impediment', 'fa fa-bookmark');

INSERT INTO work_items (id, type, space_id, fields, number, created_at)
VALUES
    (current_setting('wi1.id')::uuid, current_setting('wit.id')::uuid, current_setting('sp1.id')::uuid, '{"system.title":"WI1"}'::json, 1, '2018-09-17 16:01'),
    (current_setting('wi2.id')::uuid, current_setting('wit.id')::uuid, current_setting('sp1.id')::uuid, '{"system.title":"WI2"}'::json, 2, '2018-09-17 18:01'),
    (current_setting('wi3.id')::uuid, current_setting('wit.id')::uuid, current_setting('sp2.id')::uuid, '{"system.title":"WI3"}'::json, 1, '2018-09-17 12:01'),
    (current_setting('wi4.id')::uuid, current_setting('wit.id')::uuid, current_setting('sp2.id')::uuid, '{"system.title":"WI4"}'::json, 2, '2018-09-17 17:01');

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
