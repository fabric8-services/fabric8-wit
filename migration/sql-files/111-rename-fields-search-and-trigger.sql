
----------------------------------------------------------------------------------------------------
--- Copied from 082-iteration-related-changed.sql
----------------------------------------------------------------------------------------------------

-- looks very similar to step 071 but here we replace the 
-- `WHERE i.id::text = NEW.Fields->>'system_iteration'` comparison with 
-- `WHERE i.id = (NEW.Fields->>'system_iteration')::uuid` to use the 
-- index of iterations on the `id` column (primary key)
drop trigger if exists workitem_link_iteration_trigger on work_items;
drop function if exists iteration_set_relationship_timestamp_on_workitem_linking();
drop trigger if exists workitem_unlink_iteration_trigger on work_items;
drop function if exists iteration_set_relationship_timestamp_on_workitem_unlinking();
drop trigger if exists workitem_soft_delete_trigger on work_items;
drop function if exists iteration_set_relationship_timestamp_on_workitem_deletion();

-- trigger and function when a workitem is linked to an iteration
CREATE FUNCTION iteration_set_relationship_timestamp_on_workitem_linking() RETURNS trigger AS $$
    -- trigger to fill the `relationships_changed_at` column when an interation is set
    BEGIN
        UPDATE iterations i SET relationships_changed_at = NEW.updated_at 
        WHERE NEW.Fields->>'system_iteration' IS NOT NULL AND i.id = (NEW.Fields->>'system_iteration')::uuid;
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER workitem_link_iteration_trigger AFTER UPDATE ON work_items 
    FOR EACH ROW
    WHEN (NEW.deleted_at IS NULL 
        -- only occurs when the `system_iteration` field changed to a non-null value
        AND NEW.Fields->>'system_iteration' IS NOT NULL 
        AND (OLD.Fields->>'system_iteration' IS NULL OR NEW.Fields->>'system_iteration' != OLD.Fields->>'system_iteration'))
    EXECUTE PROCEDURE iteration_set_relationship_timestamp_on_workitem_linking();

-- trigger and function when an iteration is unset for a workitem 
CREATE FUNCTION iteration_set_relationship_timestamp_on_workitem_unlinking() RETURNS trigger AS $$
    -- trigger to fill the `relationships_changed_at` column when an interation is set
    BEGIN
        UPDATE iterations i SET relationships_changed_at = NEW.updated_at 
        WHERE OLD.Fields->>'system_iteration' IS NOT NULL AND i.id = (OLD.Fields->>'system_iteration')::uuid;
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER workitem_unlink_iteration_trigger AFTER UPDATE ON work_items 
    FOR EACH ROW
    WHEN (OLD.deleted_at IS NULL 
        -- only occurs when the `system_iteration` field was a non-null value before, and then it changed
        AND OLD.Fields->>'system_iteration' IS NOT NULL 
        AND (NEW.Fields->>'system_iteration' IS NULL OR NEW.Fields->>'system_iteration'!= OLD.Fields->>'system_iteration'))
    EXECUTE PROCEDURE iteration_set_relationship_timestamp_on_workitem_unlinking();

-- trigger and function when a workitem that is soft-deleted was linked to an iteration
CREATE FUNCTION iteration_set_relationship_timestamp_on_workitem_deletion() RETURNS trigger AS $$
    -- trigger to fill the `relationships_changed_at` column when an interation is set
    BEGIN
        UPDATE iterations i SET relationships_changed_at = NEW.deleted_at 
        WHERE OLD.Fields->>'system_iteration' IS NOT NULL AND i.id = (OLD.Fields->>'system_iteration')::uuid;
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER workitem_soft_delete_trigger AFTER UPDATE ON work_items 
    FOR EACH ROW
    WHEN (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL)
    EXECUTE PROCEDURE iteration_set_relationship_timestamp_on_workitem_deletion();

----------------------------------------------------------------------------------------------------
--- Taken from 065-workitem-id-unique-per-space.sql
----------------------------------------------------------------------------------------------------


-- UPDATE the 'tsv' COLUMN with the text value of the existing 'content' 
-- element in the 'system_description' JSON document
UPDATE work_items SET tsv =
    setweight(to_tsvector('english', "number"::text),'A') ||
    setweight(to_tsvector('english', coalesce(fields->>'system_title','')),'B') ||
    setweight(to_tsvector('english', coalesce(fields#>>'{system_description, content}','')),'C');

-- fill the 'tsv' COLUMN with the text value of the created/modified 'content' 
-- element in the 'system_description' JSON document
DROP trigger IF EXISTS upd_tsvector on work_items;
DROP FUNCTION IF EXISTS workitem_tsv_TRIGGER();

CREATE FUNCTION workitem_tsv_TRIGGER() RETURNS TRIGGER AS $$
begin
  new.tsv :=
    setweight(to_tsvector('english', new.number::text),'A') ||
    setweight(to_tsvector('english', coalesce(new.fields->>'system_title','')),'B') ||
    setweight(to_tsvector('english', coalesce(new.fields#>>'{system_description, content}','')),'C');
  return new;
end
$$ LANGUAGE plpgsql; 

CREATE TRIGGER upd_tsvector BEFORE INSERT OR UPDATE OF number, fields ON work_items
FOR EACH ROW EXECUTE PROCEDURE workitem_tsv_TRIGGER();