CREATE TABLE labels (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    name text NOT NULL CHECK(name <> ''),
    color text NOT NULL,
    space_id uuid NOT NULL REFERENCES spaces (id) ON DELETE CASCADE,
    CONSTRAINT labels_name_space_id_unique UNIQUE(space_id, name)
);

CREATE INDEX label_name_idx ON labels USING btree (name);
