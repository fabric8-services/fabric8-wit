-- Create index on the work_item_id column
CREATE INDEX ix_workitem_id ON work_item_revisions USING BTREE (work_item_id);