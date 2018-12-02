ALTER TABLE tracker_queries ADD COLUMN work_item_type_id uuid REFERENCES work_item_types(id);
