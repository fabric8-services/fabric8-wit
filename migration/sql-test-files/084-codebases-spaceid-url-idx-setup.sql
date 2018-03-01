-- Create space for this test
INSERT INTO
    spaces(id, name)
VALUES
    ('86af5178-9b41-469b-9096-57e5155c3f31', 'foobarspace');

-- Insert in codebase an entry for spaceid and url
INSERT INTO
    codebases(url, space_id)
VALUES
    ('https://github.com/fabric8-services/fabric8-wit/',
     '86af5178-9b41-469b-9096-57e5155c3f31');

-- Repeat the above sql command to create duplicate entry
INSERT INTO
    codebases(url, space_id)
VALUES
    ('https://github.com/fabric8-services/fabric8-wit/',
     '86af5178-9b41-469b-9096-57e5155c3f31');

-- Repeat the above sql command to create duplicate entry
INSERT INTO
    codebases(url, space_id)
VALUES
    ('https://github.com/fabric8-services/fabric8-wit/',
     '86af5178-9b41-469b-9096-57e5155c3f31');
