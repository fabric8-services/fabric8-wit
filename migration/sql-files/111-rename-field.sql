DO 
	$do$
	declare k text;
	BEGIN
		-- update json field keys of work_item_types
                FOR k in SELECT jsonb_object_keys(Fields) FROM work_item_types LOOP
			UPDATE work_item_types SET fields = replace(fields::TEXT, k, regexp_replace(k, '\.', '_'))::jsonb; 
		END LOOP;

		-- update json field keys of work_items
                FOR k in SELECT jsonb_object_keys(Fields) FROM work_items LOOP
			UPDATE work_items SET fields = replace(fields::TEXT, k, regexp_replace(k, '\.', '_'))::jsonb; 
		END LOOP;
	END
	$do$;

