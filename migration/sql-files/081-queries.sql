CREATE TABLE queries (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    title text NOT NULL CHECK(title <> ''),
    fields jsonb NOT NULL,
    space_id uuid NOT NULL REFERENCES spaces (id) ON DELETE CASCADE,
    creator uuid NOT NULL
);
CREATE UNIQUE INDEX queries_title_space_id_creator_unique ON queries (title, space_id, creator) WHERE deleted_at IS NULL;

CREATE INDEX query_creator_idx ON queries USING btree (creator);
