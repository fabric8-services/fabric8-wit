-- wrong color code
-- this should fail
DELETE FROM labels;
DELETE FROM spaces where id='11111111-7171-0000-0000-000000000000';
INSERt INTO spaces (id, name) VALUES ('11111111-7171-0000-0000-000000000000', 'test space');
INSERT INTO labels (name, text_color, background_color, space_id) VALUES ('some name', '#2f4c56', '#rrsstt', '11111111-7171-0000-0000-000000000000');
