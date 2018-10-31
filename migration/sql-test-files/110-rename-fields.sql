insert into space_templates (id, name)
values
  (
    '9aa3a541-4c01-4f1e-a53d-9473f0e7e573',
    'test template'
  );
insert into spaces (id, name, space_template_id)
values
  (
    '8706684a-f1df-421d-adc7-f5ebc0733dcf',
    'test space', '9aa3a541-4c01-4f1e-a53d-9473f0e7e573'
  );
-- create work item type
INSERT INTO work_item_types (id, name, space_template_id, fields)
VALUES
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
    '{"system.title":"Work item 1", "system.number":1234}' :: json
  );
