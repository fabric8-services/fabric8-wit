-- Create the number sequences table
CREATE TABLE number_sequences (
    space_id uuid REFERENCES spaces(id) ON DELETE CASCADE,
    table_name text CHECK (trim(table_name::text) <> ''),
    current_val INTEGER NOT NULL,
    PRIMARY KEY (space_id, table_name)
);

-- Copy over all numbers from the old table
INSERT INTO number_sequences (space_id, table_name, current_val)
    (SELECT space_id, 'work_items', current_val FROM work_item_number_sequences);

-- Delete old number table
DELETE FROM work_item_number_sequences;