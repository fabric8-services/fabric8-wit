CREATE OR REPLACE FUNCTION limit_execution_order_to_space() RETURNS void as $$
-- limit_execution_order_to_space() function limits order to space
	DECLARE
		i integer;
		r RECORD;
		a RECORD;
		workitems_cursor CURSOR FOR SELECT id, execution_order, space_id from work_items;
		spaces_cursor CURSOR FOR SELECT id from spaces;
	BEGIN
		open spaces_cursor;
			FOR a in FETCH ALL FROM spaces_cursor
				LOOP
					i:=1000;
					open workitems_cursor;
						FOR r in FETCH ALL FROM workitems_cursor
							LOOP
								UPDATE work_items set execution_order=i where id=r.id AND space_id=a.id;
								i := i+1000;
							END LOOP;
					close workitems_cursor;
				END LOOP;
		close spaces_cursor;

	END $$ LANGUAGE plpgsql;

DO $$ BEGIN
	PERFORM limit_execution_order_to_space();
	DROP FUNCTION limit_execution_order_to_space();
END $$;
