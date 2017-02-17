CREATE INDEX area_path_btree_index ON areas USING BTREE (path);
CREATE INDEX area_path_gist_index ON areas USING GIST (path);
