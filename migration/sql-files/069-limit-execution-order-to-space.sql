-- limit_execution_order_to_space() function limits execution order of workitems to space
-- Fetch space from spaces. For workitems which belong to that space, set their execution order sequentially -> 1000, 2000, 3000 and so on.
-- Fetch next space from spaces, for each workitem which belongs to that space, again set their execution_order sequentially -> 1000, 2000, 3000...
CREATE OR REPLACE FUNCTION limit_execution_order_to_space() RETURNS void as $$
	DECLARE
		i integer;
		r RECORD;
		a RECORD;
		workitems_cursor CURSOR FOR SELECT id, execution_order, space_id from work_items;
		spaces_cursor CURSOR FOR SELECT id from spaces;
	BEGIN
		open spaces_cursor;
			FOR a in FETCH ALL FROM spaces_cursor -- fetch space from spaces
				LOOP
					i:=1000;
					open workitems_cursor;
						FOR r in FETCH ALL FROM workitems_cursor -- fetch workitem from work_items
							LOOP
								-- for workitems belonging to that space, set their execution order sequentially 1000, 2000, 3000
								UPDATE work_items set execution_order=i where id=r.id AND space_id=a.id;
								i = i+1000;
							END LOOP;
					close workitems_cursor;
				END LOOP;
		close spaces_cursor;

	END $$ LANGUAGE plpgsql;

DO $$ BEGIN
	PERFORM limit_execution_order_to_space();
	DROP FUNCTION limit_execution_order_to_space();
END $$;
