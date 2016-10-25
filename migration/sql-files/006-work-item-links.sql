-- Here's the layout I'm trying to create:
-- (NOTE: work_items and work_item_types tables already exist)
--        
--           .----------------.
--           | work_items     |        .-----------------.
--           | ----------     |        | work_item_types |
--     .------>id bigserial   |        | --------------- |
--     |     | type text ------------>>| name text       |
--     |     | [other fields] |    |   | [other fields]  |
--     |     '----------------'    |   '-----------------'
--     |                           |
--     |   .------------------.    |   .-----------------------.
--     |   | work_item_links  |    |   | work_item_link_types  |
--     |   | ---------------  |    |   | ---------             |
--     |   | id uuid          | .------> id uuid               |
--     .-----source_id bigint | |  |   | name text             |
--      '----target_id bigint |/   |   | description text      |
--         | link_type_id uuid|    '-----source_type_name text |
--         '------------------'     '----target_type_name text |
--                                     | forward_name text     |
--                                     | reverse_name text     |
--    .--------------------------.    .- link_category_id uuid |
--    |work_item_link_categories |   / '-----------------------'
--    |------------------------- |  /
--    | id uuid                 <---
--    | name text                |
--    | description text         |
--    '--------------------------'

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- work item link categories

CREATE TABLE work_item_link_categories (
    created_at  timestamp with time zone,
    updated_at  timestamp with time zone,
    deleted_at  timestamp with time zone DEFAULT NULL,

    id          uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    version     integer,

    name        text NOT NULL UNIQUE,
    description text 
);

-- work item link types

CREATE TYPE work_item_link_topology AS ENUM ('network', 'directed_network', 'dependency', 'tree');

CREATE TABLE work_item_link_types (
    created_at          timestamp with time zone,
    updated_at          timestamp with time zone,
    deleted_at          timestamp with time zone DEFAULT NULL,
    
    id                  uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    version             integer,

    name                text NOT NULL,
    description         text,
    source_type_name    text REFERENCES work_item_types(name) NOT NULL,
    target_type_name    text REFERENCES work_item_types(name) NOT NULL,
    forward_name        text NOT NULL, -- MUST not be NULL because UI needs this
    reverse_name        text NOT NULL, -- MUST not be NULL because UI needs this
    topology            work_item_link_topology NOT NULL, 
    link_category_id    uuid REFERENCES work_item_link_categories(id) NOT NULL
);

-- work item links

CREATE TABLE work_item_links (
    created_at      timestamp with time zone,
    updated_at      timestamp with time zone,
    deleted_at      timestamp with time zone DEFAULT NULL,
    
    id              uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    version         integer,

    link_type_id    uuid REFERENCES work_item_link_types(id) NOT NULL,
    source_id       bigint REFERENCES work_items(id) NOT NULL,
    target_id       bigint REFERENCES work_items(id) NOT NULL
);