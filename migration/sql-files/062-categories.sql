CREATE TABLE categories (
	created_at timestamp with time zone,
	updated_at timestamp with time zone,
	deleted_at timestamp with time zone,
	id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
	name text NOT NULL
);

CREATE INDEX categories_id_index ON categories (id);
CREATE UNIQUE INDEX categories_name_idx ON categories (name) WHERE deleted_at IS NULL;

CREATE TABLE work_item_type_categories (
	created_at timestamp with time zone,
	updated_at timestamp with time zone,
	deleted_at timestamp with time zone,
	id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
	category_id uuid NOT NULL,
	work_item_type_id uuid NOT NULL
);

CREATE INDEX work_item_type_categories_id_idx ON work_item_type_categories (id);
CREATE UNIQUE INDEX work_item_type_categories_idx ON work_item_type_categories (category_id, work_item_type_id) WHERE deleted_at IS NULL;
ALTER TABLE work_item_type_categories ADD CONSTRAINT work_item_type_id_work_item_types_id_fk FOREIGN KEY (work_item_type_id) REFERENCES work_item_types (id) ON DELETE CASCADE;
ALTER TABLE work_item_type_categories ADD CONSTRAINT category_id_categories_id_fk FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE CASCADE;
