DELETE FROM spaces where id='171d52ff-fa00-46d7-ac24-94269908ad7a';
INSERT INTO spaces (id, name) VALUES ('171d52ff-fa00-46d7-ac24-94269908ad7a', 'test space');

-- empty query name
-- this should fail
DELETE FROM queries;
INSERT INTO queries(fields, space_id, creator) 
	VALUES ('{"assignee": "me"}', '171d52ff-fa00-46d7-ac24-94269908ad7a', '5ff348cf-57bc-4411-8812-21840107d25c');

