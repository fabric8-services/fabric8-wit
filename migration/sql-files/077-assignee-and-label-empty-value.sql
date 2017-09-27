update work_items set fields=jsonb_set(fields, '{system.assignees}', '[]') where Fields->>'system.assignees' is null;
update work_items set fields=jsonb_set(fields, '{system.labels}', '[]') where Fields->>'system.labels' is null;
