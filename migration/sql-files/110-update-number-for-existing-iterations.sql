-- Assign a number to every existing iteration partitioned by their space and in
-- ascending creation order.
UPDATE iterations SET number = seq.row_number
FROM (
    SELECT id, space_id, created_at, row_number() OVER (PARTITION BY space_id ORDER BY created_at ASC)
    FROM iterations
) AS seq
WHERE iterations.id = seq.id;

-- Make "number" a required column and add an index for faster querying over
-- "space_id" and "number".
ALTER TABLE iterations ALTER COLUMN number SET NOT NULL;
ALTER TABLE iterations ADD CONSTRAINT iterations_space_id_number_idx UNIQUE (space_id, number);

-- Update the "number_sequences" table with the maximum for iterations.
INSERT INTO number_sequences (space_id, table_name, current_val)
    SELECT space_id, 'iterations' "table_name", MAX(number)
    FROM iterations
    WHERE number IS NOT NULL
    GROUP BY 1,2;