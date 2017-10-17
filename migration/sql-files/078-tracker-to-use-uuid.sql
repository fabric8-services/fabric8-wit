-- Add a new UUID field to the trackers table and let other tables use that instead of the current id
ALTER TABLE trackers ADD COLUMN tracker_id uuid DEFAULT uuid_generate_v4() NOT NULL;

-- "Change type" of tracker_id column in tracker_items table
ALTER TABLE tracker_items ADD COLUMN tracker_id_new uuid;
UPDATE tracker_items SET tracker_id_new = trackers.tracker_id FROM trackers WHERE trackers.id = tracker_items.tracker_id;
ALTER TABLE tracker_items ALTER COLUMN tracker_id_new SET NOT NULL;
ALTER TABLE tracker_items DROP COLUMN tracker_id CASCADE;
ALTER TABLE tracker_items RENAME COLUMN tracker_id_new TO tracker_id;

-- "Change type" of tracker_id column in tacker_queries table
ALTER TABLE tracker_queries ADD COLUMN tracker_id_new uuid;
UPDATE tracker_queries SET tracker_id_new = trackers.tracker_id FROM trackers WHERE trackers.id = tracker_queries.tracker_id;
ALTER TABLE tracker_queries ALTER COLUMN tracker_id_new SET NOT NULL;
ALTER TABLE tracker_queries DROP COLUMN tracker_id CASCADE;

-- "Rename" primary key of trackers table
ALTER TABLE tracker_queries RENAME COLUMN tracker_id_new TO tracker_id;
ALTER TABLE trackers DROP COLUMN id CASCADE;
ALTER TABLE trackers RENAME COLUMN tracker_id TO id;
ALTER TABLE trackers ADD CONSTRAINT trackers_pkey PRIMARY KEY (id);

-- Set new foreign keys in tracker_items and tracker_queries to use new UUID field
ALTER TABLE ONLY tracker_items ADD CONSTRAINT tracker_items_tracker_id_trackers_id_foreign FOREIGN KEY (tracker_id) REFERENCES trackers(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE ONLY tracker_queries ADD CONSTRAINT tracker_queries_tracker_id_trackers_id_foreign FOREIGN KEY (tracker_id) REFERENCES trackers(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
