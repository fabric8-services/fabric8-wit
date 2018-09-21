CREATE OR REPLACE FUNCTION rename_wi_fields() RETURNS void as $$
    DECLARE
        k text;
    BEGIN
		-- update json field keys of work_item_types
                FOR k in SELECT jsonb_object_keys(Fields) FROM work_item_types LOOP
			UPDATE work_item_types SET fields = fields - k || jsonb_build_object(regexp_replace(k, '\.', '_'), fields->k); 
		END LOOP;

		-- update json field keys of work_items
                FOR k in SELECT jsonb_object_keys(Fields) FROM work_items LOOP
			UPDATE work_items SET fields = fields - k || jsonb_build_object(regexp_replace(k, '\.', '_'), fields->k); 
		END LOOP;
END $$ LANGUAGE plpgsql;

DO $$ BEGIN
    PERFORM rename_wi_fields();
    DROP FUNCTION rename_wi_fields();
END $$;

