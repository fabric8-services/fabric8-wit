SET user1.id = '{{index . 0}}';
SET identity.id = '{{index . 1}}';
SET sp_template.id = '{{index . 2}}';
SET space.id = '{{index . 3}}';
SET work_item_type.id = '{{index . 4}}';
SET work_item.id = '{{index . 5}}';
SET work_item_revision.id = '{{index . 6}}';

insert into users (id, email) values(current_setting('user1.id')::uuid, 'foo@bar.com');
insert into identities (id, user_id) values(current_setting('identity.id')::uuid, current_setting('user1.id')::uuid);

insert into space_templates (id, name) values(current_setting('sp_template.id')::uuid, 'test template');

insert into spaces (id, name, space_template_id) values(current_setting('space.id')::uuid, 'test space', current_setting('sp_template.id')::uuid);

-- create work item type
insert into work_item_types (id, name, space_template_id, fields)
values
(
  current_setting('work_item_type.id')::uuid,
  'foo',
  current_setting('sp_template.id')::uuid,
  '{
    "foo.bar": {"Type": {"Kind": "string"}},
    "system.area": {"Type": {"Kind": "area"}},
    "system.order": {"Type": {"Kind": "float"}}
  }'
);

-- create a work item
insert into work_items (id, type, space_id, fields)
values
  (
    current_setting('work_item.id')::uuid,
    current_setting('work_item_type.id')::uuid,
    current_setting('space.id')::uuid,
    '{"system.title":"Work item 1", "system.number":1234, "foo.bar": 123}'::json
  );

-- create a revision
insert into work_item_revisions (id, revision_type, modifier_id, work_item_type_id, work_item_id, work_item_fields)
values
(
  current_setting('work_item_revision.id')::uuid,
  '1',
  current_setting('identity.id')::uuid,
  current_setting('work_item_type.id')::uuid,
  current_setting('work_item.id')::uuid,
  '{"system.title":"Work item 1", "system.number":1234, "foo.bar": 123}'::json
);
