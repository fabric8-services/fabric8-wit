-- empty label name
-- this should fail
INSERT INTO
	labels(created_at, updated_at, id, color, space_id)
VALUES
   (
	now(), now(), 'c0c3b254-c780-4a21-b00d-241a29e6be51', '#000000', '5c12842c-61ce-4481-b33d-163d09a732c4'
   );


