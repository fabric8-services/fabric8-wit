CREATE INDEX IF NOT EXISTS fields_title ON work_items USING GIN (to_tsvector('english', fields->>'system.title'));
CREATE INDEX IF NOT EXISTS fields_description ON work_items USING GIN (to_tsvector('english', fields->>'system.description'));
