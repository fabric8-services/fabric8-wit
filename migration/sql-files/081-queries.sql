CREATE TABLE queries (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    title text NOT NULL CHECK(title <> ''),
    fields jsonb,
    space_id uuid NOT NULL REFERENCES spaces (id) ON DELETE CASCADE,
    CONSTRAINT queries_title_space_id_unique UNIQUE(space_id, title)
);

CREATE INDEX query_title_idx ON queries USING btree (title);
