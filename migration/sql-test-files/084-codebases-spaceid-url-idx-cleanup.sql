-- Delete entries from codebases
DELETE FROM codebases
WHERE
    (url = 'https://github.com/fabric8-services/fabric8-wit/' AND
    space_id = '48d8987f-d7c2-454f-b67c-4cad74199b26');

-- Delete entries from spaces
DELETE FROM spaces
WHERE
    id = '48d8987f-d7c2-454f-b67c-4cad74199b26';
