-- add an index on work item links to find existing links where a given work item 
-- is the target
CREATE INDEX work_item_links_target_id_idx ON work_item_links USING btree (target_id, deleted_at);