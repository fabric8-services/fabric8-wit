-- add a foreign key from spaces.owner_id to identities.id
alter table spaces add constraint spaces_owner_id_identities_id_fk foreign key (owner_id) REFERENCES identities (id);