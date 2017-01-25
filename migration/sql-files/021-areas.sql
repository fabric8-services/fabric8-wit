CREATE EXTENSION IF NOT EXISTS "ltree";

CREATE TABLE areas (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    space_id uuid,
    path ltree,
    name text
);

CREATE INDEX ax_space_id ON areas USING btree (space_id);