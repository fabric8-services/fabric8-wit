-- Delete entries from codebases
DELETE FROM codebases
WHERE
    (url = 'https://github.com/fabric8-services/fabric8-wit/' AND
    space_id = '86af5178-9b41-469b-9096-57e5155c3f31');

-- Delete entries from spaces
DELETE FROM spaces
WHERE
    id = '86af5178-9b41-469b-9096-57e5155c3f31';
