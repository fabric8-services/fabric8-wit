
-- create space template
INSERT INTO space_templates (id,name,description) VALUES('c87fa296-e02b-416e-b5ce-4833a66fa6d4', 'test space template 3', 'test template');

-- create space
insert into spaces (id,name,space_template_id) values ('bcbb4e5a-7500-4a65-9019-0dd8e19f261b', 'test space 3', 'c87fa296-e02b-416e-b5ce-4833a66fa6d4');

-- create root iteration
insert into iterations (id, name, path, space_id) values ('4f6f8bed-263a-4643-97c3-4cc861337ed7', 'test iteration 1', '', 'bcbb4e5a-7500-4a65-9019-0dd8e19f261b');

-- create child iteration
insert into iterations (id, name, path, space_id) values ('f7918e5f-f998-4852-987e-135fa565503b', 'test child iteration', '4f6f8bed_263a_4643_97c3_4cc861337ed7', 'bcbb4e5a-7500-4a65-9019-0dd8e19f261b');