DROP TABLE space_related_locks;
CREATE TABLE space_related_locks(space_id uuid PRIMARY KEY REFERENCES spaces(id) ON DELETE CASCADE);

INSERT INTO space_related_locks (SELECT id from spaces);


CREATE OR REPLACE FUNCTION fill_space_related_locks() RETURNS TRIGGER AS
$BODY$
BEGIN
    INSERT INTO space_related_locks(space_id) VALUES(new.id);
    RETURN new;
END;
$BODY$
language plpgsql;

CREATE TRIGGER fill_space_related_locks_trigger
    AFTER INSERT ON spaces
    FOR EACH ROW
    EXECUTE PROCEDURE fill_space_related_locks();