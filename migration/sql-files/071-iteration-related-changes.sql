ALTER TABLE iterations ADD COLUMN relationships_changed_at timestamp with time zone;
COMMENT ON COLUMN iterations.relationships_changed_at IS 'see triggers on the ''work_items'' table''.';

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
        UPDATE iterations i SET relationships_changed_at = NEW.updated_at WHERE i.id::text = NEW.Fields->>'system.iteration';
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER workitem_link_iteration_trigger AFTER UPDATE ON work_items 
    FOR EACH ROW
    WHEN (NEW.deleted_at IS NULL 
        -- only occurs when the `system.iteration` field changed to a non-null value
        AND NEW.Fields->>'system.iteration' IS NOT NULL 
        AND (OLD.Fields->>'system.iteration' IS NULL OR NEW.Fields->>'system.iteration' != OLD.Fields->>'system.iteration'))
    EXECUTE PROCEDURE iteration_set_relationship_timestamp_on_workitem_linking();

-- trigger and function when an iteration is unset for a workitem 
CREATE FUNCTION iteration_set_relationship_timestamp_on_workitem_unlinking() RETURNS trigger AS $$
    -- trigger to fill the `relationships_changed_at` column when an interation is set
    BEGIN
        UPDATE iterations i SET relationships_changed_at = NEW.updated_at WHERE i.id::text = OLD.Fields->>'system.iteration';
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER workitem_unlink_iteration_trigger AFTER UPDATE ON work_items 
    FOR EACH ROW
    WHEN (OLD.deleted_at IS NULL 
        -- only occurs when the `system.iteration` field was a non-null value before, and then it changed
        AND OLD.Fields->>'system.iteration' IS NOT NULL 
        AND (NEW.Fields->>'system.iteration' IS NULL OR NEW.Fields->>'system.iteration'!= OLD.Fields->>'system.iteration'))
    EXECUTE PROCEDURE iteration_set_relationship_timestamp_on_workitem_unlinking();

-- trigger and function when a workitem that is soft-deleted was linked to an iteration
CREATE FUNCTION iteration_set_relationship_timestamp_on_workitem_deletion() RETURNS trigger AS $$
    -- trigger to fill the `relationships_changed_at` column when an interation is set
    BEGIN
        UPDATE iterations i SET relationships_changed_at = NEW.deleted_at WHERE i.id::text = OLD.Fields->>'system.iteration';
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER workitem_soft_delete_trigger AFTER UPDATE ON work_items 
    FOR EACH ROW
    WHEN (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL)
    EXECUTE PROCEDURE iteration_set_relationship_timestamp_on_workitem_deletion();


