-- Change foreign key from "identities" to "users" to cascade.
ALTER TABLE identities
    ADD FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    DROP CONSTRAINT identities_user_id_users_id_fk;

-- Change foreign key from "comment_revisions" to "identities" to ON DELETE CASCADE.
ALTER TABLE comment_revisions
    ADD FOREIGN KEY (modifier_id) REFERENCES identities(id) ON DELETE CASCADE,
    DROP CONSTRAINT comment_revisions_identity_fk;

-- Change foreign key from "work_item_link_revisions" to "identities" to ON DELETE CASCADE.
ALTER TABLE work_item_link_revisions
    ADD FOREIGN KEY (modifier_id) REFERENCES identities(id) ON DELETE CASCADE,
    DROP CONSTRAINT work_item_link_revisions_modifier_id_fk;

-- Change foreign key from "work_item_revisions" to "identities" to ON DELETE CASCADE.
ALTER TABLE work_item_revisions
    ADD FOREIGN KEY (modifier_id) REFERENCES identities(id) ON DELETE CASCADE,
    DROP CONSTRAINT work_item_revisions_identity_fk;