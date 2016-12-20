ALTER TABLE projects RENAME TO spaces;
ALTER INDEX projects_name_idx RENAME TO spaces_name_idx;
ALTER TABLE iterations RENAME COLUMN project_id to space_id;