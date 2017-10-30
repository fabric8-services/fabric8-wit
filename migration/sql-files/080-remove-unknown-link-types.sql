SET LOCAL linktypes.bug_blocker = '{{index . 0}}';
SET LOCAL linktypes.related = '{{index . 1}}';
SET LOCAL linktypes.parenting = '{{index . 2}}';

SET LOCAL linkcats.systemcat = '{{index . 3}}';
SET LOCAL linkcats.usercat = '{{index . 4}}';

-- The following link types exist in the current production database but they
-- are not known to the code and therefore we re-assign all links associated to
-- those link types with their appropriate link type and later remove these
-- unknown link types.
SET LOCAL linktypes.unknown_bug_blocker = 'aad2a4ad-d601-4104-9804-2c977ca2e0c1';
SET LOCAL linktypes.unknown_related = '355b647b-adc5-46b3-b297-cc54bc0554e6';
SET LOCAL linktypes.unknown_parenting = '7479a9b9-8607-46fa-9535-d448fa8768ab';

-- Re-assign any existing link to correct type but avoid those links that
-- already exist.
UPDATE work_item_links l 
    SET link_type_id = current_setting('linktypes.bug_blocker')::uuid 
    WHERE link_type_id = current_setting('linktypes.unknown_bug_blocker')::uuid
    AND NOT EXISTS(
        SELECT * FROM work_item_links
        WHERE
            link_type_id = current_setting('linktypes.bug_blocker')::uuid
            AND source_id = l.source_id
            AND target_id = l.target_id
        );
UPDATE work_item_links l 
    SET link_type_id = current_setting('linktypes.related')::uuid 
    WHERE link_type_id = current_setting('linktypes.unknown_related')::uuid
    AND NOT EXISTS(
        SELECT * FROM work_item_links
        WHERE
            link_type_id = current_setting('linktypes.related')::uuid
            AND source_id = l.source_id
            AND target_id = l.target_id
        );
UPDATE work_item_links l 
    SET link_type_id = current_setting('linktypes.parenting')::uuid 
    WHERE link_type_id = current_setting('linktypes.unknown_parenting')::uuid
    AND NOT EXISTS(
        SELECT * FROM work_item_links
        WHERE
            link_type_id = current_setting('linktypes.parenting')::uuid
            AND source_id = l.source_id
            AND target_id = l.target_id
        );

-- Update revisions
UPDATE work_item_link_revisions rev SET
    work_item_link_type_id = (SELECT link_type_id FROM work_item_links WHERE id = rev.work_item_link_id);

-- Remove unknown link categories
DELETE FROM work_item_link_categories WHERE id NOT IN (
     current_setting('linkcats.systemcat')::uuid,
     current_setting('linkcats.usercat')::uuid
);

-- Finally, delete old link types
DELETE FROM work_item_link_types WHERE id NOT IN (
    current_setting('linktypes.bug_blocker')::uuid,
    current_setting('linktypes.related')::uuid,
    current_setting('linktypes.parenting')::uuid
);

-- Add foreign keys to revisions table
ALTER TABLE work_item_link_revisions  ADD CONSTRAINT link_rev_link_type_fk  FOREIGN KEY (work_item_link_type_id) REFERENCES work_item_link_types(id) ON DELETE CASCADE;
ALTER TABLE work_item_link_revisions ADD CONSTRAINT link_rev_source_id_fk FOREIGN KEY (work_item_link_source_id) REFERENCES work_items(id) ON DELETE CASCADE;
ALTER TABLE work_item_link_revisions ADD CONSTRAINT link_rev_target_id_fk FOREIGN KEY (work_item_link_target_id) REFERENCES work_items(id) ON DELETE CASCADE;