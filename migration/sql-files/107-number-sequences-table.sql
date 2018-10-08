 -- Create the number_sequences table
CREATE TABLE number_sequences (
    space_id uuid REFERENCES spaces(id) ON DELETE CASCADE,
    table_name text CHECK (trim(table_name::text) <> ''),
    current_val INTEGER NOT NULL,
    PRIMARY KEY (space_id, table_name)
);