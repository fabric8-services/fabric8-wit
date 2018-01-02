CREATE TABLE queries (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    title text NOT NULL CHECK(title <> ''),
    fields jsonb NOT NULL,
    space_id uuid NOT NULL REFERENCES spaces (id) ON DELETE CASCADE,
    creator uuid NOT NULL,
    CONSTRAINT queries_title_space_id_creator_unique UNIQUE(title, space_id, creator, deleted_at)
);

CREATE INDEX query_creator_idx ON queries USING btree (creator);
