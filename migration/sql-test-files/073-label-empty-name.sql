-- empty label name
-- this should fail
delete from labels;
INSERT INTO labels(text_color, background_color, space_id) 
	VALUES ('#fff9db', '#2f4c56', '5c12842c-61ce-4481-b33d-163d09a732c4');

