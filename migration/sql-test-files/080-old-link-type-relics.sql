SET spaces.system = '{{index . 0}}';

SET linktypes.bug_blocker = '{{index . 1}}';
SET linktypes.related = '{{index . 2}}';
SET linktypes.parenting = '{{index . 3}}';
-- we don't create this link type it just symbolizes a link type with no link type in existence
SET linktypes.completely_unknown = 'd30bb732-b277-48de-8d76-db241878bd30';

SET linktypes.unknown_bug_blocker = 'aad2a4ad-d601-4104-9804-2c977ca2e0c1';
SET linktypes.unknown_related = '355b647b-adc5-46b3-b297-cc54bc0554e6';
SET linktypes.unknown_parenting = '7479a9b9-8607-46fa-9535-d448fa8768ab';

SET cats.system = '{{index . 4}}';
SET cats.unknown_system = '75bc23dc-5aa3-4b1a-a3a6-b315e7ebeaa0';
SET cats.usercat = '{{index . 5}}';
SET cats.unknown_usercat = 'f83073d9-b79e-471b-a9a4-68248dd431ab';

INSERT INTO spaces (id, name) VALUES 
    (current_setting('spaces.system')::uuid, 'system.space')
    ON CONFLICT DO NOTHING;

INSERT INTO work_item_link_categories (id, name) VALUES
    (current_setting('cats.system')::uuid, 'system'),
    (current_setting('cats.unknown_system')::uuid, 'another system link category'),
    (current_setting('cats.usercat')::uuid, 'user'),
    (current_setting('cats.unknown_usercat')::uuid, 'another user link category')
    ON CONFLICT DO NOTHING;

INSERT INTO work_item_link_types (id, name, forward_name, reverse_name, topology, link_category_id, space_id) VALUES
    -- These are the known link types
    (   current_setting('linktypes.bug_blocker')::uuid,
        'Bug blocker', 'blocks', 'blocked by', 'network',
        current_setting('cats.system')::uuid, 
        current_setting('spaces.system')::uuid),

    (   current_setting('linktypes.related')::uuid, 
        'Related planner item', 'relates to', 'is related to', 'network', 
        current_setting('cats.system')::uuid, 
        current_setting('spaces.system')::uuid),

    (   current_setting('linktypes.parenting')::uuid,
        'Parent child item', 'parent of', 'child of', 'tree',
        current_setting('cats.system')::uuid,
        current_setting('spaces.system')::uuid),

    -- -- Insert the link types that exist in production but are left-overs from
    -- -- commit 90c595eaa02bde744207b6699d40ae4cc34a834e when I introduced fixed
    -- -- IDs for link types and categories. The following link types should be
    -- -- removed when we migrate to version 78 of the database.
    (   current_setting('linktypes.unknown_bug_blocker')::uuid,
        'Bug blocker', 'blocks', 'blocked by', 'network',
        current_setting('cats.unknown_system')::uuid, 
        current_setting('spaces.system')::uuid),

    (   current_setting('linktypes.unknown_related')::uuid, 
        'Related planner item', 'relates to', 'is related to', 'network', 
        current_setting('cats.unknown_system')::uuid, 
        current_setting('spaces.system')::uuid),

    (   current_setting('linktypes.unknown_parenting')::uuid,
        'Parent child item', 'parent of', 'child of', 'tree',
        current_setting('cats.unknown_system')::uuid,
        current_setting('spaces.system')::uuid);

-- Create some work items

SET wits.test = 'd998e454-08f7-48cb-97a0-c985073e77e2';
SET wis.parent1 = '95375720-4c50-4244-bd7c-04a7a33c4f28';
SET wis.parent2 = '4ab532d5-17fe-43e4-9c91-b62dddc3a02a';
SET wis.child1 = 'e7c3fab3-00a8-4ab8-9401-3545a92d5daa';
SET wis.child2 = '27d6c8e7-57a1-4dfa-8b05-3f2e3af9f5ca';

INSERT INTO work_item_types (id, name, space_id) VALUES
    (current_setting('wits.test')::uuid, 'Test WIT', current_setting('spaces.system')::uuid);

INSERT INTO work_items (id, space_id, type, fields) VALUES
    (current_setting('wis.parent1')::uuid, current_setting('spaces.system')::uuid, current_setting('wits.test')::uuid, '{"system.title":"Parent"}'::json),
    (current_setting('wis.parent2')::uuid, current_setting('spaces.system')::uuid, current_setting('wits.test')::uuid, '{"system.title":"Parent"}'::json),
    (current_setting('wis.child1')::uuid, current_setting('spaces.system')::uuid, current_setting('wits.test')::uuid, '{"system.title":"Child"}'::json),
    (current_setting('wis.child2')::uuid, current_setting('spaces.system')::uuid, current_setting('wits.test')::uuid, '{"system.title":"Child"}'::json);

-- Create links using both link types

SET ids.link1 = 'f87eb5dd-5021-4af1-9c60-79923dfa7ebd';
SET ids.link2 = '80f64aed-ecfe-45d7-b521-125d65ba4544';
SET ids.link3 = '77e25f85-77ff-4d53-a413-efc729a416e8';
SET ids.link4 = '45c02d6e-8196-4397-b1bd-ed354e978d4d';

INSERT INTO work_item_links (id, link_type_id, source_id, target_id) VALUES
    -- Create two links between the same WIs only using the old and new
    -- parenting link type. These links will be merged into one when migrating
    -- to version 78.
    (current_setting('ids.link1')::uuid, current_setting('linktypes.unknown_parenting')::uuid, current_setting('wis.parent1')::uuid, current_setting('wis.child1')::uuid),
    (current_setting('ids.link2')::uuid, current_setting('linktypes.parenting')::uuid, current_setting('wis.parent1')::uuid, current_setting('wis.child1')::uuid),
    -- This one will be changed to the new link type
    (current_setting('ids.link3')::uuid, current_setting('linktypes.unknown_parenting')::uuid, current_setting('wis.parent2')::uuid, current_setting('wis.child2')::uuid),
    -- This one exists because we need to create a link revision pointing to a
    -- valid link but using an unknown link type. 
    (current_setting('ids.link4')::uuid, current_setting('linktypes.related')::uuid, current_setting('wis.parent1')::uuid, current_setting('wis.child1')::uuid);

SET ids.user1 = 'e312b89c-0407-4fbb-b907-11d2ec37feec';
SET ids.user2 = 'ce5db17f-a047-41cf-b309-a64e9f293f4b';
SET ids.identity1 = '5f4f9360-c084-4039-8b94-7ea03e4d8fe1';
SET ids.identity2 = '42947554-229e-4f47-8880-ad0cad128da8';

SET modifierids.createid = '1';

-- users
INSERT INTO users(created_at, updated_at, id, email, full_name, image_url, bio, url, context_information)
VALUES (now(), now(), current_setting('ids.user1')::uuid, 'foobar1@example.com', 'test1', 'https://www.gravatar.com/avatar/testtwo2', 'my test bio one', 'http://example.com/001', '{"key": "value"}'),
(now(), now(), current_setting('ids.user2')::uuid, 'foobar2@example.com', 'test2', 'http://https://www.gravatar.com/avatar/testtwo3', 'my test bio two', 'http://example.com/002', '{"key": "value"}');

-- identities
INSERT INTO identities(created_at, updated_at, id, username, provider_type, user_id, profile_url)
VALUES (now(), now(), current_setting('ids.identity1')::uuid, 'test1', 'github', current_setting('ids.user1')::uuid, 'http://example-github.com/00123'),
(now(), now(), current_setting('ids.identity2')::uuid, 'test2', 'rhhd', current_setting('ids.user2')::uuid, 'http://example-rhd.com/00234');

-- Create appropriate link revisions for the create event
INSERT INTO work_item_link_revisions (
    revision_type, 
    modifier_id, 
    work_item_link_id, 
    work_item_link_version,
    work_item_link_type_id,
    work_item_link_source_id,
    work_item_link_target_id)
SELECT 
    current_setting('modifierids.createid')::int, 
    current_setting('ids.identity1')::uuid,
    id AS work_item_link_id,
    version AS work_item_link_version,
    link_type_id AS work_item_link_type_id,
    source_id AS work_item_link_source_id,
    target_id AS work_item_link_target_id
FROM work_item_links;

-- Manually update a link revision to point to a link type that doesn't exist.
UPDATE work_item_link_revisions SET work_item_link_type_id = current_setting('linktypes.completely_unknown')::uuid WHERE work_item_link_id = current_setting('ids.link4')::uuid;
