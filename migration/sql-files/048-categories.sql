CREATE TABLE categories (
	created_at timestamp with time zone,
	updated_at timestamp with time zone,
	deleted_at timestamp with time zone,
	id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
	name text
);

ALTER TABLE work_item_types ADD COLUMN categories_id uuid;
ALTER TABLE work_item_types ADD CONSTRAINT work_item_types_categories_fkey FOREIGN KEY (categories_id) REFERENCES categories(id);

INSERT INTO categories(name) VALUES('requirements');
INSERT INTO categories(name) VALUES('issues');
