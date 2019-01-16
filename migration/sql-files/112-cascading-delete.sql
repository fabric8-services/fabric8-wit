-- Change foreign key from "identities" to "users" to cascade.
ALTER TABLE identities
    ADD FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    DROP CONSTRAINT identities_user_id_users_id_fk;

-- Remove foreign key from "comment_revisions" to "identities".
ALTER TABLE comment_revisions
    DROP CONSTRAINT comment_revisions_identity_fk;

-- Remove foreign key from "work_item_link_revisions" to "identities".
ALTER TABLE work_item_link_revisions
    DROP CONSTRAINT work_item_link_revisions_modifier_id_fk;

-- Remove foreign key from "work_item_revisions" to "identities".
ALTER TABLE work_item_revisions
    DROP CONSTRAINT work_item_revisions_identity_fk;