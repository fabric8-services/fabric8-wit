ALTER TABLE labels ADD COLUMN border_color TEXT NOT NULL DEFAULT '#000000' CHECK(border_color ~ '^#[A-Fa-f0-9]{6}$');
