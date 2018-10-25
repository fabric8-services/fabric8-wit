-- Delete no longer needed work item link categories that is no longer used
-- since migration 106.
DROP TABLE work_item_link_categories;

-- Add missing foreign key constraint from comment to work item.
DELETE FROM comments c WHERE parent_id IS NOT NULL AND NOT EXISTS (SELECT * FROM work_items w WHERE c.parent_id = w.id);
ALTER TABLE comments ADD FOREIGN KEY (parent_id) REFERENCES work_items(id) ON DELETE CASCADE;

CREATE OR REPLACE FUNCTION archive_record()
RETURNS TRIGGER AS $$
BEGIN
    -- archive_record() can be use used as the trigger function on all tables
    -- that want to archive their data into a separate *_archive table after
    -- it was (soft-)DELETEd on the main table. The function will have no effect
    -- if it is being used on a non-DELETE or non-UPDATE trigger.
    --
    -- You should set up a trigger like so:
    --
    --        CREATE TRIGGER soft_delete_countries
    --            AFTER
    --                -- this is what is triggered by GORM
    --                UPDATE OF deleted_at 
    --                -- this is what is triggered by a cascaded DELETE or a direct hard-DELETE
    --                OR DELETE
    --            ON countries
    --            FOR EACH ROW
    --            EXECUTE PROCEDURE archive_record();
    --
    -- The effect of such a trigger is that your entry will be archived under
    -- these circumstances:
    --
    --   1. a soft-delete happens by setting a row's `deleted_at` field to a non-`NULL` value,
    --   2. a hard-DELETE happens,
    --   3. or a cascaded DELETE happens that was triggered by one of the before mentioned events.
    --
    -- The only requirements are:
    --
    --  1. your table has a `deleted_at` field
    --  2. your table has an archive table with the extact same name and an `_archive` suffix
    --  3. your table has a primary key called `id`
    --
    -- You should set up your archive table like so:
    --
    --      CREATE TABLE your_table_archive (CHECK(deleted_at IS NOT NULL)) INHERITS(your_table);
    
    -- When a soft-delete happens
    IF (TG_OP = 'UPDATE' AND NEW.deleted_at IS NOT NULL) THEN
        EXECUTE format('DELETE FROM %I.%I WHERE id = $1', TG_TABLE_SCHEMA, TG_TABLE_NAME) USING OLD.id;
        RETURN OLD;
    END IF;
    -- When a hard-DELETE or a cascaded delete happens
    IF (TG_OP = 'DELETE') THEN
        -- Set the time when the deletion happen (if not already done)
        IF (OLD.deleted_at IS NULL) THEN
            OLD.deleted_at := timenow();
        END IF;
        EXECUTE format('INSERT INTO %I.%I SELECT $1.*', TG_TABLE_SCHEMA, TG_TABLE_NAME || '_archive')
        USING OLD;
    END IF;
    RETURN NULL;
END;
  $$ LANGUAGE plpgsql;

-- Create archive tables
CREATE TABLE areas_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (areas);
CREATE TABLE codebases_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (codebases);
CREATE TABLE comments_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (comments);
CREATE TABLE identities_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (identities);
CREATE TABLE iterations_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (iterations);
CREATE TABLE labels_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (labels);
CREATE TABLE space_templates_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (space_templates);
CREATE TABLE spaces_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (spaces);
CREATE TABLE tracker_items_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (tracker_items);
CREATE TABLE tracker_queries_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (tracker_queries);
CREATE TABLE trackers_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (trackers);
CREATE TABLE users_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (users);
CREATE TABLE work_item_board_columns_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (work_item_board_columns);
CREATE TABLE work_item_boards_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (work_item_boards);
CREATE TABLE work_item_child_types_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (work_item_child_types);
CREATE TABLE work_item_link_types_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (work_item_link_types);
CREATE TABLE work_item_links_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (work_item_links);
CREATE TABLE work_item_type_group_members_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (work_item_type_group_members);
CREATE TABLE work_item_type_groups_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (work_item_type_groups);
CREATE TABLE work_item_types_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (work_item_types);
CREATE TABLE work_items_archive (CHECK (deleted_at IS NOT NULL)) INHERITS (work_items);

-- Setup triggers
CREATE TRIGGER archive_areas AFTER UPDATE OF deleted_at OR DELETE ON areas FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_codebases AFTER UPDATE OF deleted_at OR DELETE ON codebases FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_comments AFTER UPDATE OF deleted_at OR DELETE ON comments FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_identities AFTER UPDATE OF deleted_at OR DELETE ON identities FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_iterations AFTER UPDATE OF deleted_at OR DELETE ON iterations FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_labels AFTER UPDATE OF deleted_at OR DELETE ON labels FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_space_templates AFTER UPDATE OF deleted_at OR DELETE ON space_templates FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_spaces AFTER UPDATE OF deleted_at OR DELETE ON spaces FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_tracker_items AFTER UPDATE OF deleted_at OR DELETE ON tracker_items FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_tracker_queries AFTER UPDATE OF deleted_at OR DELETE ON tracker_queries FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_trackers AFTER UPDATE OF deleted_at OR DELETE ON trackers FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_users AFTER UPDATE OF deleted_at OR DELETE ON users FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_work_item_board_columns AFTER UPDATE OF deleted_at OR DELETE ON work_item_board_columns FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_work_item_boards AFTER UPDATE OF deleted_at OR DELETE ON work_item_boards FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_work_item_child_types AFTER UPDATE OF deleted_at OR DELETE ON work_item_child_types FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_work_item_link_types AFTER UPDATE OF deleted_at OR DELETE ON work_item_link_types FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_work_item_links AFTER UPDATE OF deleted_at OR DELETE ON work_item_links FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_work_item_type_group_members AFTER UPDATE OF deleted_at OR DELETE ON work_item_type_group_members FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_work_item_type_groups AFTER UPDATE OF deleted_at OR DELETE ON work_item_type_groups FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_work_item_types AFTER UPDATE OF deleted_at OR DELETE ON work_item_types FOR EACH ROW EXECUTE PROCEDURE archive_record();
CREATE TRIGGER archive_work_items AFTER UPDATE OF deleted_at OR DELETE ON work_items FOR EACH ROW EXECUTE PROCEDURE archive_record();

-- Archive all deleted records
DELETE FROM areas WHERE deleted_at IS NOT NULL;
DELETE FROM codebases WHERE deleted_at IS NOT NULL;
DELETE FROM comments WHERE deleted_at IS NOT NULL;
DELETE FROM identities WHERE deleted_at IS NOT NULL;
DELETE FROM iterations WHERE deleted_at IS NOT NULL;
DELETE FROM labels WHERE deleted_at IS NOT NULL;
DELETE FROM space_templates WHERE deleted_at IS NOT NULL;
DELETE FROM spaces WHERE deleted_at IS NOT NULL;
DELETE FROM tracker_items WHERE deleted_at IS NOT NULL;
DELETE FROM tracker_queries WHERE deleted_at IS NOT NULL;
DELETE FROM trackers WHERE deleted_at IS NOT NULL;
DELETE FROM users WHERE deleted_at IS NOT NULL;
DELETE FROM work_item_board_columns WHERE deleted_at IS NOT NULL;
DELETE FROM work_item_boards WHERE deleted_at IS NOT NULL;
DELETE FROM work_item_child_types WHERE deleted_at IS NOT NULL;
DELETE FROM work_item_link_types WHERE deleted_at IS NOT NULL;
DELETE FROM work_item_links WHERE deleted_at IS NOT NULL;
DELETE FROM work_item_type_group_members WHERE deleted_at IS NOT NULL;
DELETE FROM work_item_type_groups WHERE deleted_at IS NOT NULL;
DELETE FROM work_item_types WHERE deleted_at IS NOT NULL;
DELETE FROM work_items WHERE deleted_at IS NOT NULL;