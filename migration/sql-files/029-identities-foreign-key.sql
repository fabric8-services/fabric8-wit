-- Refactor Identities: add a foreign key constraint
alter table identities add constraint identies_user_id_users_id_foreign foreign key (user_id) REFERENCES users (id);
