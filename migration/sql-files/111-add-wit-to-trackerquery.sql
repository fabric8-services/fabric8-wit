CREATE OR REPLACE FUNCTION adds_wit() RETURNS void as $$
-- adds_wit() function adds work_item_type to existing tracker_queries in database
	DECLARE 
		r RECORD;
		c CURSOR FOR SELECT id, space_id, work_item_type_id from tracker_queries;
	BEGIN
		open c;
			FOR r in FETCH ALL FROM c LOOP
					UPDATE tracker_queries as tq
						SET work_item_type_id = wit.id
						FROM work_item_types as wit, spaces as sp
						WHERE
							tq.space_id = sp.id
						        AND sp.space_template_id = wit.space_template_id
						        AND wit.can_construct = true;
			END LOOP;
		close c;
END $$ LANGUAGE plpgsql;

DO $$ BEGIN
	ALTER TABLE tracker_queries ADD COLUMN work_item_type_id uuid REFERENCES work_item_types(id) ON DELETE CASCADE;
	PERFORM adds_wit();
	DROP FUNCTION adds_wit();
	ALTER TABLE tracker_queries ALTER COLUMN work_item_type_id set not null;
END $$;

