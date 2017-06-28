insert into comments (id, parent_id, body) values ('00000066-0000-0000-0000-000000000000', '00000066-0000-0000-0000-000000000000', 'a foo comment');

insert into comment_revisions (id, revision_type, modifier_id, comment_id, comment_parent_id, comment_body) 
    values ('00000066-0000-0000-0000-000000000000', 1, 'cafebabe-0000-0000-0000-000000000000', '00000066-0000-0000-0000-000000000000',  '00000065-0000-0000-0000-000000000000', 'a foo comment');