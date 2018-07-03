-- create space template
INSERT INTO space_templates (id,name,description) VALUES('2b542bba-e131-412a-8762-88688e142311', 'test space template 2b542bba-e131-412a-8762-88688e142311', 'test template');

-- create space
insert into spaces (id,name,space_template_id) values ('b12cf363-3625-4b7b-99f9-159c49720200', 'test space b12cf363-3625-4b7b-99f9-159c49720200', '2b542bba-e131-412a-8762-88688e142311');

-- create work item type Impediment
DELETE FROM work_item_types WHERE id = '03b9bb64-4f65-4fa7-b165-494cd4f01401';
INSERT INTO work_item_types (id,name,can_construct,space_template_id,fields,description,icon) VALUES ('03b9bb64-4f65-4fa7-b165-494cd4f01401', 'Impediment', 'false', '2b542bba-e131-412a-8762-88688e142311', '{"resolution": {"type": {"values": ["Done", "Rejected", "Duplicate", "Incomplete Description", "Can not Reproduce", "Partially Completed", "Deferred", "Wont Fix", "Out of Date", "Explained", "Verified"], "base_type": {"kind": "string"}, "simple_type": {"kind": "enum"}, "rewritable_values": false}, "label": "Resolution", "required": false, "read_only": false, "description": "The reason why this work items state was last changed.\n"}, "system.area": {"type": {"kind": "area"}, "label": "Area", "required": false, "read_only": false, "description": "The area to which the work item belongs"}, "system.order": {"type": {"kind": "float"}, "label": "Execution Order", "required": false, "read_only": true, "description": "Execution Order of the workitem"}, "system.state": {"type": {"values": ["New", "Open", "In Progress", "Resolved", "Closed"], "base_type": {"kind": "string"}, "simple_type": {"kind": "enum"}, "rewritable_values": false}, "label": "State", "required": true, "read_only": false, "description": "The state of the impediment."}, "system.title": {"type": {"kind": "string"}, "label": "Title", "required": true, "read_only": false, "description": "The title text of the work item"}, "system.labels": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "label"}}, "label": "Labels", "required": false, "read_only": false, "description": "List of labels attached to the work item"}, "system.number": {"type": {"kind": "integer"}, "label": "Number", "required": false, "read_only": true, "description": "The unique number that was given to this workitem within its space."}, "system.creator": {"type": {"kind": "user"}, "label": "Creator", "required": true, "read_only": false, "description": "The user that created the work item"}, "system.codebase": {"type": {"kind": "codebase"}, "label": "Codebase", "required": false, "read_only": false, "description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "user"}}, "label": "Assignees", "required": false, "read_only": false, "description": "The users that are assigned to the work item"}, "system.iteration": {"type": {"kind": "iteration"}, "label": "Iteration", "required": false, "read_only": false, "description": "The iteration to which the work item belongs"}, "system.created_at": {"type": {"kind": "instant"}, "label": "Created at", "required": false, "read_only": true, "description": "The date and time when the work item was created"}, "system.updated_at": {"type": {"kind": "instant"}, "label": "Updated at", "required": false, "read_only": true, "description": "The date and time when the work item was last updated"}, "system.description": {"type": {"kind": "markup"}, "label": "Description", "required": false, "read_only": false, "description": "A descriptive text of the work item"}, "system.remote_item_id": {"type": {"kind": "string"}, "label": "Remote item", "required": false, "read_only": false, "description": "The ID of the remote work item"}}', 'Description for Impediment', 'fa fa-bookmark');

-- Create a few work items for Impediment - one with and one without a
-- resolution set.
insert into work_items (id, type, space_id, fields) values ('24ed462d-0430-4ffe-ba4f-7b5725b6a411', '03b9bb64-4f65-4fa7-b165-494cd4f01401', 'b12cf363-3625-4b7b-99f9-159c49720200', '{"system.title":"Work item 1", "resolution":"Rejected"}'::json); -- this resolution does only exist in the former version of the agile template

insert into work_items (id, type, space_id, fields) values ('6a870ee3-e57c-4f98-9c7a-3cdf2ef5c222', '03b9bb64-4f65-4fa7-b165-494cd4f01401', 'b12cf363-3625-4b7b-99f9-159c49720200', '{"system.title":"Work item 2"}'::json); -- no resolution specified