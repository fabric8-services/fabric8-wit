-- Only drop the foreign key constraint for now so that existing replicas of old
-- code can still create link types (only happens during startup migration and
-- template import).
ALTER TABLE work_item_link_types DROP CONSTRAINT work_item_link_types_link_category_id_fkey;