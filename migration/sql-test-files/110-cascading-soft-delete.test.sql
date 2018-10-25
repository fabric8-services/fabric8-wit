SET id.area = '{{index . 0}}';
SET id.codebase = '{{index . 1}}';
SET id.comment = '{{index . 2}}';
SET id.identity = '{{index . 3}}';
SET id.iteration = '{{index . 4}}';
SET id.label = '{{index . 5}}';
SET id.space = '{{index . 6}}';
SET id.spaceTemplate = '{{index . 7}}';
SET id.tracker = '{{index . 8}}';
SET id.trackerItem = '{{index . 9}}';
SET id.trackerQuery = '{{index . 10}}';
SET id._user = '{{index . 11}}';
SET id.workItemBoardColumn = '{{index . 12}}';
SET id.workItemBoard = '{{index . 13}}';
SET id.workItemChildType = '{{index . 14}}';
SET id.workItem = '{{index . 15}}';
SET id.workItemLink = '{{index . 16}}';
SET id.workItemLinkType = '{{index . 17}}';
SET id.workItemTypeGroup = '{{index . 18}}';
SET id.workItemTypeGroupMember = '{{index . 19}}';
SET id.workItemType = '{{index . 20}}';
-- deleted items
SET id.deletedArea ='{{index . 21}}';
SET id.deletedCodebase ='{{index . 22}}';
SET id.deletedComment ='{{index . 23}}';
SET id.deletedIdentity ='{{index . 24}}';
SET id.deletedIteration ='{{index . 25}}';
SET id.deletedLabel ='{{index . 26}}';
SET id.deletedSpace ='{{index . 27}}';
SET id.deletedSpaceTemplate ='{{index . 28}}';
SET id.deletedTracker ='{{index . 29}}';
SET id.deletedTrackerItem ='{{index . 30}}';
SET id.deletedTrackerQuery ='{{index . 31}}';
SET id.deletedUser ='{{index . 32}}';
SET id.deletedWorkItemBoardColumn ='{{index . 33}}';
SET id.deletedWorkItemBoard ='{{index . 34}}';
SET id.deletedWorkItemChildType ='{{index . 35}}';
SET id.deletedWorkItem ='{{index . 36}}';
SET id.deletedWorkItemLink ='{{index . 37}}';
SET id.deletedWorkItemLinkType ='{{index . 38}}';
SET id.deletedWorkItemTypeGroup ='{{index . 39}}';
SET id.deletedWorkItemTypeGroupMember ='{{index . 40}}';
SET id.deletedWorkItemType ='{{index . 41}}';

INSERT INTO space_templates (id,name,description, deleted_at) VALUES
    (current_setting('id.spaceTemplate')::uuid, current_setting('id.spaceTemplate'), 'test template', NULL),
    (current_setting('id.deletedSpaceTemplate')::uuid, current_setting('id.deletedSpaceTemplate'), 'test template', '2018-09-17 16:01');

INSERT INTO spaces (id,name,space_template_id, deleted_at) VALUES
        (current_setting('id.space')::uuid, current_setting('id.space'), current_setting('id.spaceTemplate')::uuid, NULL),
        (current_setting('id.deletedSpace')::uuid, current_setting('id.deletedSpace'), current_setting('id.spaceTemplate')::uuid, '2018-09-17 16:01');

INSERT INTO iterations (id, name, path, space_id, number, deleted_at) VALUES
    (current_setting('id.iteration')::uuid, 'root iteration', replace(current_setting('id.iteration'), '-', '_')::ltree, current_setting('id.space')::uuid, 1, NULL),
    (current_setting('id.deletedIteration')::uuid, 'deleted iteration', replace(current_setting('id.deletedIteration'), '-', '_')::ltree, current_setting('id.space')::uuid, 2, '2018-09-17 16:01');

INSERT INTO areas (id, name, path, space_id, number, deleted_at) VALUES
        (current_setting('id.area')::uuid, 'area', replace(current_setting('id.area'), '-', '_')::ltree, current_setting('id.space')::uuid, 1, NULL),
        (current_setting('id.deletedArea')::uuid, 'area deleted', replace(current_setting('id.deletedArea'), '-', '_')::ltree, current_setting('id.space')::uuid, 2, '2018-09-17 16:01');

INSERT INTO labels (id, name, text_color, background_color, space_id, deleted_at) VALUES
    (current_setting('id.label')::uuid, 'some label', '#ffffff', '#000000', current_setting('id.space')::uuid, NULL),
    (current_setting('id.deletedLabel')::uuid, 'deleted label', '#000000', '#ffffff', current_setting('id.space')::uuid, '2018-09-17 16:01');

INSERT INTO work_item_types (id, name, space_template_id, fields, description, icon, deleted_at) VALUES
    (current_setting('id.workItemType')::uuid, 'WIT1', current_setting('id.spaceTemplate')::uuid, '{"system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}}', 'Description for WIT1', 'fa fa-bookmark', NULL),
    (current_setting('id.deletedWorkItemType')::uuid, 'WIT2 Deleted', current_setting('id.spaceTemplate')::uuid, '{"system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}}', 'Description for WIT2 Deleted', 'fa fa-bookmark', '2018-09-17 16:01');

INSERT INTO work_items (id, type, space_id, fields, deleted_at) VALUES
    (current_setting('id.workItem')::uuid, current_setting('id.workItemType')::uuid, current_setting('id.space')::uuid, '{"system.title":"Work item 1"}'::json, NULL),
    (current_setting('id.deletedWorkItem')::uuid, current_setting('id.workItemType')::uuid, current_setting('id.space')::uuid, '{"system.title":"Work item 2 Deleted"}'::json, '2018-09-17 16:01');

INSERT INTO work_item_link_types (id, name, forward_name, reverse_name, topology, space_template_id, deleted_at) VALUES
    (current_setting('id.workItemLinkType')::uuid, 'Bug blocker', 'blocks', 'blocked by', 'network', current_setting('id.spaceTemplate')::uuid, NULL),
    (current_setting('id.deletedWorkItemLinkType')::uuid, 'Dependency', 'depends on', 'is dependent on', 'dependency', current_setting('id.spaceTemplate')::uuid, '2018-09-17 16:01');

INSERT INTO work_item_links (id, link_type_id, source_id, target_id, deleted_at) VALUES
    (current_setting('id.workItemLink')::uuid, current_setting('id.workItemLinkType')::uuid, current_setting('id.workItem')::uuid, current_setting('id.workItem')::uuid, NULL),
    (current_setting('id.deletedWorkItemLink')::uuid, current_setting('id.workItemLinkType')::uuid, current_setting('id.workItem')::uuid, current_setting('id.workItem')::uuid, '2018-09-17 16:01');

INSERT INTO comments (id, parent_id, body, deleted_at) VALUES
    (current_setting('id.comment')::uuid, current_setting('id.workItem')::uuid, 'a comment', NULL),
    (current_setting('id.deletedComment')::uuid, current_setting('id.workItem')::uuid, 'another comment', '2018-09-17 16:01');

INSERT INTO codebases (id, space_id, type, url, stack_id, deleted_at) VALUES
    (current_setting('id.codebase')::uuid, current_setting('id.space')::uuid, 'git', 'git@github.com:fabric8-services/fabric8-wit.git', 'golang-default', NULL),
    (current_setting('id.deletedCodebase')::uuid, current_setting('id.space')::uuid, 'git', 'git@github.com:fabric8-services/fabric8-common.git', 'golang-default', '2018-09-17 16:01');

INSERT INTO users (id, email, full_name, deleted_at) VALUES
    (current_setting('id._user')::uuid, concat('john_doe@', current_setting('id._user'), '.com'), 'John Doe', NULL),
    (current_setting('id.deletedUser')::uuid, concat('jane_doe@', current_setting('id.deletedUser'), '.com'), 'Jane Doe', '2018-09-17 16:01');

INSERT INTO identities (id, username, user_id, deleted_at) VALUES
    (current_setting('id.identity')::uuid, current_setting('id.identity'), current_setting('id._user')::uuid, NULL),
    (current_setting('id.deletedIdentity')::uuid, current_setting('id.deletedIdentity'), current_setting('id.deletedUser')::uuid, '2018-09-17 16:01');

INSERT INTO trackers (id, url, type, deleted_at) VALUES
    (current_setting('id.tracker')::uuid, 'https://api.github.com/', 'github', NULL),
    (current_setting('id.deletedTracker')::uuid, 'https://api.github.com/', 'github', '2018-09-17 16:01');

INSERT INTO tracker_items (id, remote_item_id, item, tracker_id, deleted_at) VALUES
    (current_setting('id.trackerItem')::bigint, '1234', '5678', current_setting('id.tracker')::uuid, NULL),
    (current_setting('id.deletedTrackerItem')::bigint, '1234', '5678', current_setting('id.tracker')::uuid, '2018-09-17 16:01');

INSERT INTO tracker_queries (id, query, schedule, space_id, tracker_id, deleted_at) VALUES
    (current_setting('id.trackerQuery')::bigint, 'after', 'the', current_setting('id.space')::uuid, current_setting('id.tracker')::uuid, NULL),
    (current_setting('id.deletedTrackerQuery')::bigint, 'before', 'the', current_setting('id.space')::uuid, current_setting('id.tracker')::uuid, '2018-09-17 16:01');

INSERT INTO work_item_boards (id, space_template_id, name, description, context_type, context, deleted_at) VALUES
    (current_setting('id.workItemBoard')::uuid, current_setting('id.spaceTemplate')::uuid, 'foo board', 'foo', 'my context type', 'my context', NULL),
    (current_setting('id.deletedWorkItemBoard')::uuid, current_setting('id.spaceTemplate')::uuid, 'bar board', 'bar', 'my context type', 'my context', '2018-09-17 16:01');

INSERT INTO work_item_board_columns (id, board_id, name, column_order, trans_rule_key, trans_rule_argument, deleted_at) VALUES
    (current_setting('id.workItemBoardColumn')::uuid, current_setting('id.workItemBoard')::uuid, 'foo board column', 1, 'my trans rule key', 'my trans rule argument', NULL),
    (current_setting('id.deletedWorkItemBoardColumn')::uuid, current_setting('id.workItemBoard')::uuid, 'bar board column', 2, 'my trans rule key', 'my trans rule argument', '2018-09-17 16:01');

INSERT INTO work_item_child_types (id, parent_work_item_type_id, child_work_item_type_id, deleted_at) VALUES
    (current_setting('id.workItemChildType')::uuid, current_setting('id.workItemType')::uuid, current_setting('id.workItemType')::uuid, NULL),
    (current_setting('id.deletedWorkItemChildType')::uuid, current_setting('id.workItemType')::uuid, current_setting('id.workItemType')::uuid, '2018-09-17 16:01');

INSERT INTO work_item_type_groups (id, name, bucket, space_template_id, deleted_at) VALUES
    (current_setting('id.workItemTypeGroup')::uuid, 'foo group', 'portfolio', current_setting('id.spaceTemplate')::uuid, NULL),
    (current_setting('id.deletedWorkItemTypeGroup')::uuid, 'foo group', 'portfolio', current_setting('id.spaceTemplate')::uuid, '2018-09-17 16:01');

INSERT INTO work_item_type_group_members (id, type_group_id, work_item_type_id, deleted_at) VALUES
    (current_setting('id.workItemTypeGroupMember')::uuid, current_setting('id.workItemTypeGroup')::uuid, current_setting('id.workItemType')::uuid, NULL),
    (current_setting('id.deletedWorkItemTypeGroupMember')::uuid, current_setting('id.workItemTypeGroup')::uuid, current_setting('id.workItemType')::uuid, '2018-09-17 16:01');
