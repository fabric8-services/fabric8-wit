 ALTER TABLE areas ADD CONSTRAINT areas_name_space_id_path_unique UNIQUE(space_id,name,path);
