DELETE FROM spaces where id='e2297c54-eb0c-4eb0-b1f9-d3212cda5e1f';
INSERT INTO spaces (id, name) VALUES ('e2297c54-eb0c-4eb0-b1f9-d3212cda5e1f', 'test space');

DELETE FROM queries;
INSERT INTO
	queries(title, fields, space_id, creator)
VALUES
   (
	'assigned to me', '{"key": "value"}', 
    'e2297c54-eb0c-4eb0-b1f9-d3212cda5e1f',
    '5f6d7daf-6be3-4171-a77b-857b327c4bac'
   );
INSERT INTO
	queries(title, fields, space_id, creator)
VALUES
   (
	'assigned to me', '{"key": "value"}',
    'e2297c54-eb0c-4eb0-b1f9-d3212cda5e1f',
    '5f6d7daf-6be3-4171-a77b-857b327c4bac'
   );
