SET id.area = '{{index . 0}}';
SET id.areaDeleted = '{{index . 1}}';
SET id.comment = '{{index . 2}}';
SET id.commentDeleted = '{{index . 3}}';
SET id.iter = '{{index . 4}}';
SET id.iterDeleted = '{{index . 5}}';
SET id.label = '{{index . 6}}';
SET id.labelDeleted = '{{index . 7}}';
SET id.space = '{{index . 8}}';
SET id.spaceDeleted = '{{index . 9}}';
SET id.spaceTemplate = '{{index . 10}}';
SET id.spaceTemplateDeleted = '{{index . 11}}';
SET id.workItem = '{{index . 12}}';
SET id.workItemDeleted = '{{index . 13}}';
SET id.workItemLink = '{{index . 14}}';
SET id.workItemLinkDeleted = '{{index . 15}}';
SET id.workItemLinkType = '{{index . 16}}';
SET id.workItemLinkTypeDeleted = '{{index . 17}}';
SET id.workItemType = '{{index . 18}}';
SET id.workItemTypeDeleted = '{{index . 19}}';

INSERT INTO space_templates (id,name,description, deleted_at) VALUES
    (current_setting('id.spaceTemplate')::uuid, current_setting('id.spaceTemplate'), 'test template', NULL),
    (current_setting('id.spaceTemplateDeleted')::uuid, current_setting('id.spaceTemplateDeleted'), 'test template', '2018-09-17 16:01');

INSERT INTO spaces (id,name,space_template_id, deleted_at) VALUES
        (current_setting('id.space')::uuid, current_setting('id.space'), current_setting('id.spaceTemplate')::uuid, NULL),
        (current_setting('id.spaceDeleted')::uuid, current_setting('id.spaceDeleted'), current_setting('id.spaceTemplate')::uuid, '2018-09-17 16:01');

INSERT INTO iterations (id, name, path, space_id, deleted_at) VALUES
    (current_setting('id.iterRoot')::uuid, 'root iteration', replace(current_setting('id.iter'), '-', '_')::ltree, current_setting('id.space')::uuid, NULL),
    (current_setting('id.iterDeleted')::uuid, 'deleted iteration', replace(current_setting('id.iterDeleted'), '-', '_')::ltree, current_setting('id.space')::uuid, '2018-09-17 16:01');

INSERT INTO areas (id, name, path, space_id, deleted_at) VALUES
        (current_setting('id.area')::uuid, 'area', replace(current_setting('id.area'), '-', '_')::ltree, current_setting('id.space')::uuid, NULL),
        (current_setting('id.areaDeleted')::uuid, 'area deleted', replace(current_setting('id.areaDeleted'), '-', '_')::ltree, current_setting('id.space')::uuid, '2018-09-17 16:01');

INSERT INTO labels (id, name, text_color, background_color, space_id, deleted_at) VALUES
    (current_setting('id.label')::uuid, 'some label', '#ffffff', '#000000', current_setting('id.space')::uuid, NULL),
    (current_setting('id.labelDeleted')::uuid, 'deleted label', '#000000', '#ffffff', current_setting('id.space')::uuid, '2018-09-17 16:01'),

INSERT INTO work_item_types (id, name, space_template_id, fields, description, icon, deleted_at) VALUES
    (current_setting('id.workItemType')::uuid, 'WIT1', current_setting('id.spaceTemplate')::uuid, '{"system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}}', 'Description for WIT1', 'fa fa-bookmark', NULL),
    (current_setting('id.workItemTypeDeleted')::uuid, 'WIT2 Deleted', current_setting('id.spaceTemplate')::uuid, '{"system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}}', 'Description for WIT2 Deleted', 'fa fa-bookmark', '2018-09-17 16:01');

INSERT INTO work_items (id, type, space_id, fields, deleted_at) VALUES
    (current_setting('id.workItem')::uuid, current_setting('id.workItemType')::uuid, current_setting('id.space')::uuid, '{"system.title":"Work item 1"}'::json, NULL),
    (current_setting('id.workItemDeleted')::uuid, current_setting('id.workItemType')::uuid, current_setting('id.space')::uuid, '{"system.title":"Work item 2 Deleted"}'::json, '2018-09-17 16:01');

INSERT INTO comments (id, parent_id, body, deleted_at) VALUES
    (current_setting('id.comment')::uuid, current_setting('id.workItem')::uuid, 'a comment', NULL),
    (current_setting('id.commentDeleted')::uuid, current_setting('id.workItem')::uuid, 'another comment', '2018-09-17 16:01');

INSERT INTO work_item_link_types (id, name, forward_name, reverse_name, topology, space_template_id, deleted_at) VALUES
    (current_setting('id.workItemLinkType')::uuid, 'Bug blocker', 'blocks', 'blocked by', 'network', current_setting('id.spaceTemplate')::uuid, NULL),
    (current_setting('id.workItemLinkTypeDeleted')::uuid, 'Dependency', 'depends on', 'is dependent on', 'dependency', current_setting('id.spaceTemplate')::uuid, '2018-09-17 16:01');