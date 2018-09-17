-- Create the number sequences table
CREATE TABLE number_sequences (
    space_id uuid REFERENCES spaces(id) ON DELETE CASCADE,
    table_name text CHECK (trim(table_name::text) <> ''),
    current_val INTEGER NOT NULL,
    PRIMARY KEY (space_id, table_name)
);

-- -- Copy over all numbers from the old table
-- INSERT INTO number_sequences (space_id, table_name, current_val)
--     (SELECT space_id, 'work_items', current_val FROM work_item_number_sequences);

-- Delete old number table
DROP TABLE work_item_number_sequences;

-- Update existing iterations and areas with new "number" column and fill in the
-- numbers in the order iterations and areas have been created.
ALTER TABLE iterations ADD COLUMN number INTEGER;
ALTER TABLE areas ADD COLUMN number INTEGER;

UPDATE iterations
SET number = seq.row_number
FROM
    (SELECT id, space_id, created_at, row_number() OVER (PARTITION BY space_id ORDER BY created_at ASC) FROM iterations) AS seq
WHERE iterations.id = seq.id;

UPDATE areas
    SET number = seq.row_number
FROM
    (SELECT id, space_id, created_at, row_number() OVER (PARTITION BY space_id ORDER BY created_at ASC) FROM areas) AS seq
WHERE areas.id = seq.id;

-- Make number a required column
ALTER TABLE iterations ALTER COLUMN number SET NOT NULL;
ALTER TABLE areas ALTER COLUMN number SET NOT NULL;

-- Update the number_sequences table with the maximum for each table type

INSERT INTO number_sequences (space_id, table_name, current_val)
    SELECT space_id, 'work_items' "table_name", MAX(number) 
    FROM work_items 
    WHERE number IS NOT NULL 
    GROUP BY 1,2;

INSERT INTO number_sequences (space_id, table_name, current_val)
    SELECT space_id, 'iterations' "table_name", MAX(number)
    FROM iterations
    WHERE number IS NOT NULL
    GROUP BY 1,2;

INSERT INTO number_sequences (space_id, table_name, current_val)
    SELECT space_id, 'areas' "table_name", MAX(number)
    FROM areas
    WHERE number IS NOT NULL
    GROUP BY 1,2;