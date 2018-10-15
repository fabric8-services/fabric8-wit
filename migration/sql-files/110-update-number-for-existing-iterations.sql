--- Assign a number to every existing iteration partitioned by their space and in
--- ascending creation order.
WITH iteration_numbers AS (
    SELECT *, ROW_NUMBER() OVER(PARTITION BY space_id ORDER BY created_at ASC) AS num
    FROM iterations
)
UPDATE iterations SET number = (SELECT num FROM iteration_numbers WHERE iteration_numbers.id = iterations.id);

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
