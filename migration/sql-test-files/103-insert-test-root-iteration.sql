
-- create space template
INSERT INTO space_templates (id,name,description) VALUES('0961ada9-afca-439f-865b-d5a8c6e3e9a2', 'test space template 2', 'test template');

-- create space
insert into spaces (id,name,space_template_id) values ('652f8798-ef89-49ac-81f9-4bb937a5d175', 'test space 2', '0961ada9-afca-439f-865b-d5a8c6e3e9a2');

-- create root iteration
insert into iterations (id, name, path, space_id) values ('abd93233-75c9-4419-a2e8-3c328736c443', 'test iteration', '', '652f8798-ef89-49ac-81f9-4bb937a5d175');

-- create child iteration
insert into iterations (id, name, path, space_id) values ('f7918e5f-f998-4852-987e-135fa565503b', 'test child iteration', 'abd93233_75c9_4419_a2e8_3c328736c443', '652f8798-ef89-49ac-81f9-4bb937a5d175');

-- create root area
insert into areas (id, name, path, space_id) values ('d87705c4-f367-4c3e-9069-c77f1e8c6c34', 'test area', '', '652f8798-ef89-49ac-81f9-4bb937a5d175');

-- create child area
insert into areas (id, name, path, space_id) values ('1516f95a-c546-45ca-953b-5483e63bd000', 'test child area', 'd87705c4_f367_4c3e_9069_c77f1e8c6c34', '652f8798-ef89-49ac-81f9-4bb937a5d175');