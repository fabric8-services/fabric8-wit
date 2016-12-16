-- is_type_or_subtype_of returns true if the typeName is of the given witName,
-- or a subtype according to the witPath; otherwise false is returned.
CREATE OR REPLACE FUNCTION is_type_or_subtype_of(typeName TEXT, witName TEXT, witPath TEXT)
RETURNS BOOLEAN AS $$
DECLARE cleanedType TEXT;
BEGIN
        SELECT trim(typeName, '/') INTO cleanedType;
        IF length(cleanedType) <= 0 THEN
                RETURN FALSE;
        END IF;
        RETURN (witName = cleanedType OR witPath LIKE '%/' || cleanedType || '/%');
END;
$$ LANGUAGE plpgsql;


-- Test function for the above function (will later be removed)
CREATE OR REPLACE FUNCTION
is_type_or_subtype_of_test(expected BOOLEAN, typeName TEXT, witName TEXT, witPath TEXT)
RETURNS void  AS
$$
DECLARE actual BOOLEAN;
BEGIN
        SELECT is_type_or_subtype_of(typeName, witName, witPath) INTO actual;
        RAISE NOTICE 'TESTING is_type_or_subtype_of(%, %, %)', typeName, witName, witPath;
        IF expected <> actual THEN
                RAISE EXCEPTION 'FAIL expected % but got %', expected, actual;
        END IF;
        RAISE NOTICE 'PASS';
END; 
$$ LANGUAGE plpgsql;
-- Test types and subtypes
select is_type_or_subtype_of_test(TRUE, 'foo', 'foo', '/foo');
select is_type_or_subtype_of_test(TRUE, 'foo', 'bar', '/foo/bar');
select is_type_or_subtype_of_test(TRUE, 'bar', 'bar', '/foo/bar');
select is_type_or_subtype_of_test(TRUE, 'foo', 'cake', '/foo/bar/cake');
select is_type_or_subtype_of_test(TRUE, 'bar', 'cake', '/foo/bar/cake');
select is_type_or_subtype_of_test(TRUE, 'cake', 'cake', '/foo/bar/cake');
-- Test we actually do return false sometimes
select is_type_or_subtype_of_test(FALSE, 'fo', 'cake', '/foo/bar/cake');
select is_type_or_subtype_of_test(FALSE, 'fo', 'foo', '/foo');
-- Test wrong argument with prefixed and trailing slashes
select is_type_or_subtype_of_test(FALSE, '', 'foo', '/foo');
select is_type_or_subtype_of_test(FALSE, '/', 'foo', '/foo');
select is_type_or_subtype_of_test(TRUE, '/foo', 'foo', '/foo');
select is_type_or_subtype_of_test(TRUE, '/foo/', 'foo', '/foo');
select is_type_or_subtype_of_test(TRUE, 'foo/', 'foo', '/foo');
-- We no longer need the test function
DROP FUNCTION is_type_or_subtype_of_test;



--
-- Ensure no link is rendered invalid when a work item's type is changed.
--


CREATE OR REPLACE FUNCTION check_link_type_violation() RETURNS trigger AS $$
DECLARE link work_item_links%rowtype;
DECLARE wit work_item_types%rowtype;
BEGIN
        RAISE NOTICE 'Detected work item type change on work item.';
        RAISE NOTICE 'Checking all work item links associated with work item % for potential type violations', NEW.id;
        -- Iterate over every link that is associated with this work item
        FOR link IN SELECT * FROM work_item_links
        WHERE NEW.id IN (source_id, target_id)
        LOOP
                RAISE NOTICE 'Checking work item link %', link.id;
                -- Get the work item type that might be violated
                IF link.source_id = NEW.id THEN
                        SELECT * INTO wit
                        FROM work_item_types
                        WHERE name = (
                                SELECT source_type_name
                                FROM work_item_link_types
                                WHERE id = link.link_type_id
                        );
                ELSE
                        SELECT * INTO wit
                        FROM work_item_types
                        WHERE name = (
                                SELECT target_type_name
                                FROM work_item_link_types
                                WHERE id = link.link_type_id
                        );
                END IF;

                -- Check that the new type of the work item is actually a subtype
                -- of the link's target or source WIT.
                IF is_type_or_subtype_of(NEW.type, wit.name, wit.path) = FALSE THEN
                        RAISE EXCEPTION 'New WI type renders work item link % invalid because the new type is not a subtype of %', link.id, wit.path
                        USING HINT = 'Attention when changing type of work item';
                END IF;
        END LOOP;
        RAISE NOTICE 'All work item links for work item % checked.', NEW.id;
        RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER check_link_type_violation_trigger
BEFORE UPDATE OF type
ON work_items
FOR EACH ROW
EXECUTE PROCEDURE check_link_type_violation();