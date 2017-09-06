CREATE TABLE labels (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    name text NOT NULL CHECK(name <> ''),
    text_color text NOT NULL DEFAULT '#000000' CHECK(text_color ~ '^#[A-Fa-f0-9]{6}$'),
    background_color text NOT NULL DEFAULT '#FFFFFF' CHECK(background_color ~ '^#[A-Fa-f0-9]{6}$'),
    space_id uuid NOT NULL REFERENCES spaces (id) ON DELETE CASCADE,
    version integer DEFAULT 0 NOT NULL,
    CONSTRAINT labels_name_space_id_unique UNIQUE(space_id, name)
);

CREATE INDEX label_name_idx ON labels USING btree (name);
