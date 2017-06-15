--
-- comments
--
insert into spaces (id, name) values ('11111111-6262-0000-0000-000000000000', 'test');
insert into work_item_types (id, name, space_id) values ('11111111-6262-0000-0000-000000000000', 'Test WIT','11111111-6262-0000-0000-000000000000');
insert into work_items (id, space_id, type, fields) values (62001, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 1"}'::json);
insert into work_items (id, space_id, type, fields) values (62002, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 2"}'::json);
-- remove previous comments
delete from comments;
-- add comments linked to work items above
insert into comments (id, parent_id, body, created_at) values ( '11111111-6262-0001-0000-000000000000', '62001', 'a comment', '2017-06-13 09:00:00.0000+00');
insert into comments (id, parent_id, body, created_at) values ( '11111111-6262-0002-0000-000000000000', '62002', 'a comment', '2017-06-13 09:10:00.0000+00');
-- mark the last comment as (soft) deleted 
update comments set deleted_at = '2017-06-13 09:15:00.0000+00' where id =  '11111111-6262-0002-0000-000000000000';

--
-- work item links
--
insert into work_items (id, space_id, type, fields) values (62003, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 3"}'::json);
insert into work_items (id, space_id, type, fields) values (62004, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 3"}'::json);
delete from work_item_links;
--insert into work_item_link_categories (id, version, name) values ('11111111-6262-0000-0000-000000000000', 1, 'test');
--insert into work_item_link_types (id, version, name, forward_name, reverse_name, topology, space_id, source_type_id, target_type_id) values (...);
insert into work_item_links (id, version, source_id, target_id, created_at) values ('11111111-6262-0001-0000-000000000000', 1, 62001, 62002, '2017-06-13 09:00:00.0000+00');
insert into work_item_links (id, version, source_id, target_id, deleted_at) values ('11111111-6262-0002-0000-000000000000', 1, 62003, 62004, '2017-06-13 10:00:00.0000+00');
update work_item_links set deleted_at = '2017-06-13 11:00:00.0000+00' where id = '11111111-6262-0002-0000-000000000000';


