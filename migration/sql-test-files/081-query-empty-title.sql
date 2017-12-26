-- empty query name
-- this should fail
delete from queries;
INSERT INTO queries(fields, space_id) 
	VALUES ('{"assignee": "me"}', '171d52ff-fa00-46d7-ac24-94269908ad7a');

