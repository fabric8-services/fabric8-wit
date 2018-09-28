update work_items set fields=jsonb_set(fields, '{system_iteration}', to_jsonb(subq.id::text)) 
    from (select id, space_id from iterations where path = '') AS subq
    where subq.space_id = work_items.space_id and fields->>'system_iteration' IS NULL;