-- add a foreign key from spaces.owner_id to identities.id
alter table spaces add constraint spaces_owner_id_identities_id_fk foreign key (owner_id) REFERENCES identities (id);
-- add index on spaces.name
create index idx_spaces_name on spaces (lower(name));
-- add index on identities.username
drop index idx_identities_username;
create index idx_identities_username on identities (lower(username));

