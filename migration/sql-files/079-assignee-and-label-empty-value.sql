-- Remove 'system.assignees' from the fields if the value is 'null' or '[]'
update work_items set fields=fields - '{{index . 0}}' where Fields->>'{{index . 0}}' is null;
update work_items set fields=fields - '{{index . 0}}' where Fields->>'{{index . 0}}'='[]';
-- Remove 'system.labels' from the fields if the value is 'null' or '[]'
update work_items set fields=fields - '{{index . 1}}' where Fields->>'{{index . 1}}' is null;
update work_items set fields=fields - '{{index . 1}}' where Fields->>'{{index . 1}}'='[]';
