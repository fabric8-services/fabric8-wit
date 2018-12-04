DELETE FROM tracker_queries; -- tracker_queries was not fully supported and not in working condition before this
ALTER TABLE tracker_queries ADD COLUMN work_item_type_id uuid not null REFERENCES work_item_types(id) ON DELETE CASCADE;
