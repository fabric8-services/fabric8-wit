SET LOCAL spaces.system = '{{index . 0}}';

SET LOCAL linktypes.bug_blocker = '{{index . 1}}';
SET LOCAL linktypes.related = '{{index . 2}}';
SET LOCAL linktypes.parenting = '{{index . 3}}';

SET LOCAL linktypes.unknown_bug_blocker = 'aad2a4ad-d601-4104-9804-2c977ca2e0c1';
SET LOCAL linktypes.unknown_related = '355b647b-adc5-46b3-b297-cc54bc0554e6';
SET LOCAL linktypes.unknown_parenting = '7479a9b9-8607-46fa-9535-d448fa8768ab';

SET LOCAL cats.system = '{{index . 4}}';
SET LOCAL cats.unknown_system = '75bc23dc-5aa3-4b1a-a3a6-b315e7ebeaa0';
SET LOCAL cats.usercat = '{{index . 5}}';
SET LOCAL cats.unknown_usercat = 'f83073d9-b79e-471b-a9a4-68248dd431ab';

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

SET LOCAL wits.test = 'd998e454-08f7-48cb-97a0-c985073e77e2';
SET LOCAL wis.parent1 = '95375720-4c50-4244-bd7c-04a7a33c4f28';
SET LOCAL wis.parent2 = '4ab532d5-17fe-43e4-9c91-b62dddc3a02a';
SET LOCAL wis.child1 = 'e7c3fab3-00a8-4ab8-9401-3545a92d5daa';
SET LOCAL wis.child2 = '27d6c8e7-57a1-4dfa-8b05-3f2e3af9f5ca';

INSERT INTO work_item_types (id, name, space_id) VALUES
    (current_setting('wits.test')::uuid, 'Test WIT', current_setting('spaces.system')::uuid);

INSERT INTO work_items (id, space_id, type, fields) VALUES
    (current_setting('wis.parent1')::uuid, current_setting('spaces.system')::uuid, current_setting('wits.test')::uuid, '{"system.title":"Parent"}'::json),
    (current_setting('wis.parent2')::uuid, current_setting('spaces.system')::uuid, current_setting('wits.test')::uuid, '{"system.title":"Parent"}'::json),
    (current_setting('wis.child1')::uuid, current_setting('spaces.system')::uuid, current_setting('wits.test')::uuid, '{"system.title":"Child"}'::json),
    (current_setting('wis.child2')::uuid, current_setting('spaces.system')::uuid, current_setting('wits.test')::uuid, '{"system.title":"Child"}'::json);

-- Create links using both link types

INSERT INTO work_item_links (link_type_id, source_id, target_id) VALUES
    -- Create two links between the same WIs only using the old and new
    -- parenting link type. These links will be merged into one when migrating
    -- to version 78.
    (current_setting('linktypes.unknown_parenting')::uuid, current_setting('wis.parent1')::uuid, current_setting('wis.child1')::uuid),
    (current_setting('linktypes.parenting')::uuid, current_setting('wis.parent1')::uuid, current_setting('wis.child1')::uuid),
    -- This one will be changed to the new link type
    (current_setting('linktypes.unknown_parenting')::uuid, current_setting('wis.parent2')::uuid, current_setting('wis.child2')::uuid);