DELETE FROM spaces where id='171d52ff-fa00-46d7-ac24-94269908ad7a';
INSERT INTO spaces (id, name) VALUES ('171d52ff-fa00-46d7-ac24-94269908ad7a', 'test space');

-- no creator ID provided
-- this should fail
DELETE FROM queries;
INSERT INTO queries(title, fields, space_id) 
	VALUES ('my queries', '{"assignee": "me"}', '171d52ff-fa00-46d7-ac24-94269908ad7a');

DELETE FROM spaces;
DELETE FROM queries;
