
-- create space template
INSERT INTO space_templates (id,name,description) VALUES('0961ada9-afca-439f-865b-d5a8c6e3e9a2', 'test space template', 'test template');

-- create space
insert into spaces (id,name,space_template_id) values ('652f8798-ef89-49ac-81f9-4bb937a5d175', 'test space', '0961ada9-afca-439f-865b-d5a8c6e3e9a2');

-- create root iteration
insert into iterations (id, name, path, space_id) values ('abd93233-75c9-4419-a2e8-3c328736c443', 'test iteration', '', '652f8798-ef89-49ac-81f9-4bb937a5d175');