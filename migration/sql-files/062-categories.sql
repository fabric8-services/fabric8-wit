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
	workitemtype_id uuid NOT NULL
);

CREATE INDEX work_item_type_categories_id_idx ON work_item_type_categories (id);
CREATE UNIQUE INDEX work_item_type_categories_idx ON work_item_type_categories (category_id, workitemtype_id) WHERE deleted_at IS NULL;
ALTER TABLE work_item_type_categories ADD CONSTRAINT category_id_categories_id_fk FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE CASCADE;
