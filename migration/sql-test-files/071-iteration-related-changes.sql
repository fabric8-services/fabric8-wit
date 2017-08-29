insert into spaces (id, name) values ('11111111-7171-0000-0000-000000000000', 'test iteration - relationships changed at');
delete from work_items;
insert into work_item_types (id, name, space_id) values ('11111111-7171-0000-0000-000000000000', 'Test WIT','11111111-7171-0000-0000-000000000000');
insert into work_items (id, created_at, space_id, type, fields) values ('11111111-7171-0000-0000-000000000000', (CURRENT_TIMESTAMP - interval '1 hour'), '11111111-7171-0000-0000-000000000000', '11111111-7171-0000-0000-000000000000', '{"system.title":"Work item 1"}'::json);
insert into work_items (id, created_at, space_id, type, fields) values ('22222222-7171-0000-0000-000000000000', (CURRENT_TIMESTAMP - interval '1 hour'), '11111111-7171-0000-0000-000000000000', '11111111-7171-0000-0000-000000000000', '{"system.title":"Work item 2"}'::json);
insert into work_items (id, created_at, space_id, type, fields) values ('33333333-7171-0000-0000-000000000000', (CURRENT_TIMESTAMP - interval '1 hour'), '11111111-7171-0000-0000-000000000000', '11111111-7171-0000-0000-000000000000', '{"system.title":"Work item 3"}'::json);
insert into work_items (id, created_at, space_id, type, fields) values ('44444444-7171-0000-0000-000000000000', (CURRENT_TIMESTAMP - interval '1 hour'), '11111111-7171-0000-0000-000000000000', '11111111-7171-0000-0000-000000000000', '{"system.title":"Work item 4"}'::json);

delete from iterations;
insert into iterations (id, name, created_at, space_id) values ('11111111-7171-0000-0000-000000000000', 'iteration 1', CURRENT_TIMESTAMP, '11111111-7171-0000-0000-000000000000');
insert into iterations (id, name, created_at, space_id) values ('22222222-7171-0000-0000-000000000000', 'iteration 2', CURRENT_TIMESTAMP, '11111111-7171-0000-0000-000000000000');
insert into iterations (id, name, created_at, space_id) values ('33333333-7171-0000-0000-000000000000', 'iteration 3', CURRENT_TIMESTAMP, '11111111-7171-0000-0000-000000000000');
insert into iterations (id, name, created_at, space_id) values ('44444444-7171-0000-0000-000000000000', 'iteration 4', CURRENT_TIMESTAMP, '11111111-7171-0000-0000-000000000000');
insert into iterations (id, name, created_at, space_id) values ('55555555-7171-0000-0000-000000000000', 'iteration 5', CURRENT_TIMESTAMP, '11111111-7171-0000-0000-000000000000');

-- link work item 1 to iteration 1
update work_items set updated_at = (CURRENT_TIMESTAMP + interval '1 hour'), fields = '{"system.title":"Work item 1", "system.iteration":"11111111-7171-0000-0000-000000000000"}'::json where id = '11111111-7171-0000-0000-000000000000';
-- link work item 2 to iteration 2 then iteration 3
update work_items set updated_at = (CURRENT_TIMESTAMP + interval '1 hour'), fields = '{"system.title":"Work item 2", "system.iteration":"22222222-7171-0000-0000-000000000000"}'::json where id = '22222222-7171-0000-0000-000000000000';
update work_items set updated_at = (CURRENT_TIMESTAMP + interval '2 hour'), fields = '{"system.title":"Work item 2", "system.iteration":"33333333-7171-0000-0000-000000000000"}'::json where id = '22222222-7171-0000-0000-000000000000';
-- link work item 3 to iteration 4 then soft-delete the work item
update work_items set fields = '{"system.title":"Work item 3", "system.iteration":"44444444-7171-0000-0000-000000000000"}'::json, updated_at = (CURRENT_TIMESTAMP + interval '1 hour') where id = '33333333-7171-0000-0000-000000000000';
update work_items set deleted_at = (CURRENT_TIMESTAMP + interval '2 hour') where id = '33333333-7171-0000-0000-000000000000';
-- link work item 4 to iteration 5 then set another, unrelated field
update work_items set fields = '{"system.title":"Work item 4", "system.iteration":"55555555-7171-0000-0000-000000000000"}'::json, updated_at = (CURRENT_TIMESTAMP + interval '1 hour') where id = '44444444-7171-0000-0000-000000000000';
update work_items set fields = '{"system.title":"Work item 4", "system.iteration":"55555555-7171-0000-0000-000000000000", "system.description":"foo"}'::json, updated_at = (CURRENT_TIMESTAMP + interval '2 hour') where id = '44444444-7171-0000-0000-000000000000';

