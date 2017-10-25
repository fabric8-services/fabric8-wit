SET LOCAL linktypes.bug_blocker = '{{index . 0}}';
SET LOCAL linktypes.related = '{{index . 1}}';
SET LOCAL linktypes.parenting = '{{index . 2}}';

-- The following link types exist in the current production database but they
-- are not known to the code and therefore we re-assign all links associated to
-- those link types with their appropriate link type and later remove these
-- unknown link types.
SET LOCAL linktypes.unknown_bug_blocker = 'aad2a4ad-d601-4104-9804-2c977ca2e0c1';
SET LOCAL linktypes.unknown_related = '355b647b-adc5-46b3-b297-cc54bc0554e6';
SET LOCAL linktypes.unknown_parenting = '7479a9b9-8607-46fa-9535-d448fa8768ab';

-- Re-assign any existing link to correct link type
UPDATE work_item_links SET link_type_id = current_setting('linktypes.bug_blocker')::uuid WHERE link_type_id = current_setting('linktypes.unknown_bug_blocker')::uuid;
UPDATE work_item_links SET link_type_id = current_setting('linktypes.related')::uuid WHERE link_type_id = current_setting('linktypes.unknown_related')::uuid;
UPDATE work_item_links SET link_type_id = current_setting('linktypes.parenting')::uuid WHERE link_type_id = current_setting('linktypes.unknown_parenting')::uuid;

-- Remove unknown link types
DELETE FROM work_item_link_types WHERE id IN (
    current_setting('linktypes.unknown_bug_blocker')::uuid,
    current_setting('linktypes.unknown_related')::uuid,
    current_setting('linktypes.unknown_parenting')::uuid
)