insert into users (id, email) values('cf22fdcf-1ce4-41d8-81bb-ed9a920b5498', 'foo@bar.com');
insert into identities (id, user_id) values('b519ad0d-2dc6-4b50-94c6-330f91711273', 'cf22fdcf-1ce4-41d8-81bb-ed9a920b5498');

insert into space_templates (id, name) values('9aa3a541-4c01-4f1e-a53d-9473f0e7e573', 'test template');

insert into spaces (id, name, space_template_id) values('8706684a-f1df-421d-adc7-f5ebc0733dcf', 'test space', '9aa3a541-4c01-4f1e-a53d-9473f0e7e573');

-- create work item type
insert into work_item_types (id, name, space_template_id, fields)
values
(
  '16bcbe81-f72f-4aa4-85c2-bbb97b4ec75f',
  'foo',
  '9aa3a541-4c01-4f1e-a53d-9473f0e7e573',
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
    'bc763f93-e45b-4ac9-bf96-e015877109d2',
    '16bcbe81-f72f-4aa4-85c2-bbb97b4ec75f',
    '8706684a-f1df-421d-adc7-f5ebc0733dcf',
    '{"system.title":"Work item 1", "system.number":1234, "foo.bar": 123}' :: json
  );

-- create a revision
insert into work_item_revisions (id, revision_type, modifier_id, work_item_type_id, work_item_id, work_item_fields)
values
(
  '2845d9b5-862e-4d97-9448-947e902e5909',
  '1',
  'b519ad0d-2dc6-4b50-94c6-330f91711273',
  '16bcbe81-f72f-4aa4-85c2-bbb97b4ec75f',
  'bc763f93-e45b-4ac9-bf96-e015877109d2',
  '{"system.title":"Work item 1", "system.number":1234, "foo.bar": 123}' :: json
);
