create temporary table field_name_map (
    old_name text,
    new_name text
);

insert into field_name_map values
    ('system.remote_item_id',       'system_remote_item_id'),
    ('system.number',               'system_number'),
    ('system.title',                'system_title'),
    ('system.description',          'system_description'),
    ('system.description.markup',   'system_description_markup'),
    ('system.description.rendered', 'system_description_rendered'),
    ('system.state',                'system_state'),
    ('system.assignees',            'system_assignees'),
    ('system.creator',              'system_creator'),
    ('system.created_at',           'system_created_at'),
    ('system.updated_at',           'system_updated_at'),
    ('system.order',                'system_order'),
    ('system.iteration',            'system_iteration'),
    ('system.area',                 'system_area'),
    ('system.codebase',             'system_codebase'),
    ('system.labels',               'system_labels'),
    ('system.boardcolumns',         'system_boardcolumns'),
    ('system.metastate',            'system_metastate');

-- Lock table "work_item_type", "work_item", "work_item_revisions"
LOCK work_item_types IN EXCLUSIVE MODE;
LOCK work_items IN EXCLUSIVE MODE;
LOCK work_item_revisions in EXCLUSIVE MODE;

DO
$$
    DECLARE
        old_field_name text;
        item_id uuid;
        new_field_name text;
        field_count integer;
        -- we will repace system. fields in the following tables
        table_field_array text[] = ARRAY[
            ['work_item_types','fields'], ['work_items','fields'], ['work_item_revisions','work_item_fields']
        ];
        item text[];
    BEGIN
        -- for each table in the array
        FOREACH item SLICE 1 IN ARRAY table_field_array
        LOOP
            RAISE INFO '% table field rename started', item[1];
            -- For each item
            for item_id in EXECUTE format('select id from %I', item[1])
            LOOP
                RAISE INFO '  Renaming fields of % %', item[1], item_id;
                -- For each field in item
                for old_field_name in EXECUTE format('
                    select jsonb_object_keys(%s) from %I where id=''%s''', item[2], item[1], item_id
                )
                LOOP
                    -- Check if the old_field_name has to be renamed
                    select new_name into new_field_name from field_name_map where old_name=old_field_name;
                    -- field_name_map contains the key which means we have to rename
                    -- it to new_field_name
                    if new_field_name is not null then
                        -- Rename old_field_name with new_field_name
                        RAISE INFO '    Changing field name from % to %', old_field_name, new_field_name;
                        EXECUTE format('
                            update %s set %2$I = %2$I - %3$s || jsonb_build_object(%4$s, %2$I->%3$s) where id=''%5$s''',
                                item[1], item[2], quote_literal(old_field_name), quote_literal(new_field_name), item_id
                        );
                    end if;
                END LOOP;
            END LOOP;
            -- Ensure we do not have any system. fields left in the table
            Execute format('
                select count(key) from %I, lateral jsonb_each_text(%I) where key like %s', item[1], item[2], quote_literal('system.%')
            ) into field_count ;
            if field_count != 0 then
                -- Fail transaction. We have system. keys present in the table
                RAISE EXCEPTION 'System.* keys present in the % table', item[1];
            end if;
            RAISE INFO '% table field rename completed', item[1];
        END LOOP;
    END;
$$
