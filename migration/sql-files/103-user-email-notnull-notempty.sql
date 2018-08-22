-- Set email = id@example.com for users whose email is '' (empty) or ' ' (any
-- length of only white spaces) or NULL in database
UPDATE users SET email=concat(id::TEXT, '@example.com') WHERE COALESCE(trim(email), '') = '';

ALTER TABLE users ALTER COLUMN email SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT email_notempty_check CHECK (trim(email) <> '');