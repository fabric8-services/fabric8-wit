ALTER TABLE comments ADD COLUMN parent_comment_id uuid;
ALTER TABLE comments ADD CONSTRAINT comments_parent_comment_id_comment_id_fk FOREIGN KEY (parent_comment_id) REFERENCES comments (id);
