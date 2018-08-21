-- Create users with null/empty email value

INSERT INTO users (full_name, email) VALUES ('Lorem1', '');
INSERT INTO users (full_name, email) VALUES ('Lorem2', '  ');
INSERT INTO users (full_name, email) VALUES ('Lorem3', '    ');
INSERT INTO users (full_name, email) VALUES ('Lorem4', NULL);