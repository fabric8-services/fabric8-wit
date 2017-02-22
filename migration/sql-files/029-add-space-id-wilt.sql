-- Alter the tables
ALTER TABLE work_item_link_types ADD space_id uuid;
ALTER TABLE work_item_link_types ADD FOREIGN KEY (space_id) REFERENCES spaces(id);

-- Create indexes
CREATE INDEX ix_space_id ON work_item_link_types USING btree (space_id);
