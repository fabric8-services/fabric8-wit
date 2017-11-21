-- Add a new UUID field to the tracker_queries table and let other tables use that instead of the current id
ALTER TABLE tracker_queries ADD COLUMN trackerquery_id uuid DEFAULT uuid_generate_v4() NOT NULL;
ALTER TABLE tracker_queries DROP COLUMN id CASCADE;
ALTER TABLE tracker_queries RENAME COLUMN trackerquery_id TO id;
