-- Remove 'system.assignees' from the fields if the value is 'null' or '[]'
update work_items set fields=fields - 'system.assignees' where Fields->>'system.assignees' is null;
update work_items set fields=fields - 'system.assignees' where Fields->>'system.assignees'='[]';
update work_items set fields=fields - 'system.labels' where Fields->>'system.labels' is null;
update work_items set fields=fields - 'system.labels' where Fields->>'system.labels'='[]';
