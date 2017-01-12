CREATE OR REPLACE FUNCTION adds_order() RETURNS void as $$
-- adds_order() function adds order to exitisting work_items in database
	DECLARE 
		i integer=1000;
		r RECORD;
		xyz CURSOR FOR SELECT id, fields->'order' from work_items;
	BEGIN
		open xyz;
			FOR r in FETCH ALL FROM xyz LOOP
				UPDATE work_items set fields=jsonb_set(fields, '{order}', to_jsonb(i)) where id=r.id;
				i := i+1000;
			END LOOP;
		close xyz;
END $$ LANGUAGE plpgsql;

DO $$ BEGIN
	PERFORM adds_order();
	DROP FUNCTION adds_order();
END $$;
