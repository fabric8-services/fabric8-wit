-- default is 'false', works with business logic as well.
ALTER TABLE identities ADD COLUMN username_updated BOOLEAN NOT NULL;
