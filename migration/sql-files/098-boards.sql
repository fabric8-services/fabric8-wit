CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- new tables for the board and boardcolumn entities
CREATE TABLE work_item_boards (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    version integer DEFAULT 0 NOT NULL,
    space_template_id uuid REFERENCES space_templates(id) ON DELETE CASCADE,
    name text NOT NULL CHECK(name <> ''),
    description text NOT NULL CHECK(description <> ''),
    context_type text NOT NULL CHECK(context_type <> ''),
    context text NOT NULL CHECK(context <> ''),
    CONSTRAINT work_item_board_name_space_template_id_unique UNIQUE(space_template_id, name)
);

CREATE INDEX work_item_boards_space_template_uidx ON work_item_boards (space_template_id) WHERE deleted_at IS NULL;

CREATE TABLE work_item_board_columns (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    board_id uuid REFERENCES work_item_boards(id) ON DELETE CASCADE,
    name text NOT NULL CHECK(name <> ''),
    column_order integer DEFAULT 0 NOT NULL,
    trans_rule_key text NOT NULL CHECK(trans_rule_key <> ''),
    trans_rule_argument text NOT NULL CHECK(trans_rule_argument <> ''),
    CONSTRAINT work_item_board_column_name_board_id_unique UNIQUE(board_id, name)
);

CREATE INDEX work_item_board_columns_board_uidx ON work_item_board_columns (board_id) WHERE deleted_at IS NULL;
