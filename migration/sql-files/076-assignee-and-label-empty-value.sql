update work_items set fields=jsonb_set(fields, '{system.assignees}', 'null') where Fields->>'system.assignees'='[]';
update work_items set fields=jsonb_set(fields, '{system.labels}', 'null') where Fields->>'system.labels'='[]';
