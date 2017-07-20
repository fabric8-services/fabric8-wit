-- This script creates workitems in two different space

-- cleanup the table
DELETE from work_items;

-- insert two spaces
insert into spaces (id, name) values ('00000000-0000-0000-0000-000000000123', 'space 1');
insert into spaces (id, name) values ('00000000-0000-0000-0000-000000000124', 'space 2');

-- insert work item type
insert into work_item_types (id, name, space_id) values ('00000000-0000-0000-0000-000000000003', 'type 1', '00000000-0000-0000-0000-000000000123');

-- insert workitem in space 1
insert into work_items (id, type, execution_order, space_id) values ('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000003', 1000, '00000000-0000-0000-0000-000000000123');

-- insert workitem in space 1
insert into work_items (id, type, execution_order, space_id) values ('00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000003', 2000, '00000000-0000-0000-0000-000000000123');

-- insert workitem in space 2
insert into work_items (id, type, execution_order, space_id) values ('00000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000003', 3000, '00000000-0000-0000-0000-000000000124');
