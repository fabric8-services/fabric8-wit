ALTER TABLE iterations
	ADD COLUMN user_active bool DEFAULT false,
	ADD COLUMN active_status bool DEFAULT false;
