-- Add two columns to `iteration` relation
ALTER TABLE iterations ADD description TEXT;
ALTER TABLE iterations ADD version INTEGER DEFAULT 0 NOT NULL;