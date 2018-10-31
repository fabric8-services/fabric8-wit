DROP table if exists field_name_map;
create table field_name_map (
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

DO
$BODY$
    declare old_field_name text;
    declare type_id uuid;
    declare new_field_name text;
    declare field_count integer;
    BEGIN
        RAISE INFO 'Workitem Type field rename started';
        -- For each workitem type
        for type_id in select id from work_item_types order by name desc
        LOOP
            RAISE INFO 'Renaming fields of workitem type %', type_id;
            -- For each field in workitem type
            for old_field_name in select jsonb_object_keys(fields) from work_item_types where id=type_id
            LOOP
                -- Check if the old_field_name has to be renamed
                select new_name into new_field_name from field_name_map where old_name=old_field_name;
                -- field_name_map contains the key which means we have to rename
                -- it to new_field_name
                if new_field_name is not null then
                    -- Rename old_field_name with new_field_name
                    RAISE INFO 'Changing field name from % to %', old_field_name, new_field_name;
                    update work_item_types set fields = fields - old_field_name || jsonb_build_object(new_field_name, fields->old_field_name) where id=type_id;
                end if;
            END LOOP;
        END LOOP;

        -- Ensure we do not have any system. fields left in the database
        select count(key) into field_count from work_item_types, lateral jsonb_each_text(fields) where key like 'system.%';
        if field_count != 0 then
            -- Fail transaction. We have system. keys present in the database
            RAISE EXCEPTION 'System.* keys present in the work_item_types table';
        end if;
        RAISE INFO 'Workitem Type field renames completed';
    END;
$BODY$
