-- Create space for this test
INSERT INTO
    spaces(id, name)
VALUES
    ('48d8987f-d7c2-454f-b67c-4cad74199b26', 'foobarspace');


-- Insert in codebase an entry for spaceid and url that is deleted before
INSERT INTO
    codebases(url, space_id, id, deleted_at)
VALUES
    ('https://github.com/fabric8-services/fabric8-wit/',
     '48d8987f-d7c2-454f-b67c-4cad74199b26',
     '6bef707f-e2a4-4a39-95b8-8b51c4d8589d',
     '2018-03-10 05:28:33.262133+00');

-- Repeat the above sql command to create duplicate entry
-- but which is not deleted before.
INSERT INTO
    codebases(url, space_id, id)
VALUES
    ('https://github.com/fabric8-services/fabric8-wit/',
     '48d8987f-d7c2-454f-b67c-4cad74199b26',
     '97bd2bc3-106a-41d8-a1d9-3fdd6d2df1f7');

-- Insert in codebase an entry for spaceid and url that is deleted before
INSERT INTO
    codebases(url, space_id, id, deleted_at)
VALUES
    ('https://github.com/fabric8-services/fabric8-wit/',
     '48d8987f-d7c2-454f-b67c-4cad74199b26',
     '25feb91f-dbed-40ce-bb46-335a4d741213',
     '2018-03-10 05:28:33.262133+00');
