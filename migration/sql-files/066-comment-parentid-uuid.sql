-- first, we ADD a new COLUMN for the 'parent id' as a UUID in the `comments` table
ALTER TABLE comments ADD COLUMN "parent_id_uuid" UUID;
UPDATE comments SET parent_id_uuid = parent_id::uuid;
-- then drop the old 'parent_id' column and rename the new one to 'parent_id'
ALTER TABLE comments DROP COLUMN "parent_id";
ALTER TABLE comments RENAME COLUMN "parent_id_uuid" TO "parent_id";

-- second, we ADD a new COLUMN for the 'parent id' as a UUID in the `comment_revisions` table
ALTER TABLE comment_revisions ADD COLUMN "comment_parent_id_uuid" UUID;
UPDATE comment_revisions SET comment_parent_id_uuid = comment_parent_id::uuid;
-- then drop the old 'parent_id' column and rename the new one to 'parent_id'
ALTER TABLE comment_revisions DROP COLUMN "comment_parent_id";
ALTER TABLE comment_revisions RENAME COLUMN "comment_parent_id_uuid" TO "comment_parent_id";