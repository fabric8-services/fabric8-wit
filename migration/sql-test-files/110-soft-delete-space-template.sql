SET id.spaceTemplate = '{{index . 0}}';

-- This should cause all entities that reference the space template directly or indirectly to be soft-deleted as well
UPDATE space_templates SET deleted_at = '2018-09-17 16:01' WHERE id = current_setting('id.spaceTemplate')::uuid; 