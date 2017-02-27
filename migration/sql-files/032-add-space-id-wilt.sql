-- Alter the table work_item_link_types
ALTER TABLE work_item_link_types ADD space_id uuid;
ALTER TABLE work_item_link_types ADD FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE;

-- Create indexes
CREATE INDEX ix_space_id ON work_item_link_types USING btree (space_id);
