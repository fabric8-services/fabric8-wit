SET id.spaceTemplate = '{{index . 0}}';
SET id.tracker = '{{index . 1}}';
SET id._user = '{{index . 2}}';

-- This should cause all entities that reference the space template directly or
-- indirectly to be soft-deleted as well
UPDATE space_templates SET deleted_at = '2018-09-17 16:01' WHERE id = current_setting('id.spaceTemplate')::uuid; 

-- Trackers are currently not used and therefore not connected to a space or a
-- space template. That's why we deleted it manually here to have a cascaded
-- delete on tracker queries and tracker items.
UPDATE trackers SET deleted_at = '2018-09-17 16:01' WHERE id = current_setting('id.tracker')::uuid;


UPDATE users SET deleted_at = '2018-09-17 16:01' WHERE id = current_setting('id._user')::uuid;