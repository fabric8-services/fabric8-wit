-- modify the `work_item_number_sequences` to include an independant ID column for the primary key
alter table work_item_number_sequences drop constraint work_item_number_sequences_pkey;
alter table work_item_number_sequences add column  id uuid primary key DEFAULT uuid_generate_v4() NOT NULL;


