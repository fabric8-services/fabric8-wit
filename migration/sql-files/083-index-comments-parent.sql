-- add index on comments.parent_id
create index idx_comments_parentid on comments using btree (parent_id);

