
-- This removes any potentially existing effort field from all work items
-- and the work item type definition of the type theme.
UPDATE work_items SET fields=(fields - 'effort') WHERE type='5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a';
UPDATE work_item_types SET fields=(fields - 'effort') WHERE id='5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a';

-- This removes any potentially existing business_value field from all work items
-- and the work item type definition of the type theme.
UPDATE work_items SET fields=(fields - 'business_value') WHERE type='5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a';
UPDATE work_item_types SET fields=(fields - 'business_value') WHERE id='5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a';

-- This removes any potentially existing time_criticality field from all work items
-- and the work item type definition of the type theme.
UPDATE work_items SET fields=(fields - 'time_criticality') WHERE type='5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a';
UPDATE work_item_types SET fields=(fields - 'time_criticality') WHERE id='5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a';

-- This removes any potentially existing effort field from all work items
-- and the work item type definition of the type epic.
UPDATE work_items SET fields=(fields - 'effort') WHERE type='2c169431-a55d-49eb-af74-cc19e895356f';
UPDATE work_item_types SET fields=(fields - 'effort') WHERE id='2c169431-a55d-49eb-af74-cc19e895356f';

-- This removes any potentially existing business_value field from all work items
-- and the work item type definition of the type epic.
UPDATE work_items SET fields=(fields - 'business_value') WHERE type='2c169431-a55d-49eb-af74-cc19e895356f';
UPDATE work_item_types SET fields=(fields - 'business_value') WHERE id='2c169431-a55d-49eb-af74-cc19e895356f';

-- This removes any potentially existing time_criticality field from all work items
-- and the work item type definition of the type epic.
UPDATE work_items SET fields=(fields - 'time_criticality') WHERE type='2c169431-a55d-49eb-af74-cc19e895356f';
UPDATE work_item_types SET fields=(fields - 'time_criticality') WHERE id='2c169431-a55d-49eb-af74-cc19e895356f';

-- This removes any potentially existing component field from all work items
-- and the work item type definition of the type epic.
UPDATE work_items SET fields=(fields - 'component') WHERE type='2c169431-a55d-49eb-af74-cc19e895356f';
UPDATE work_item_types SET fields=(fields - 'component') WHERE id='2c169431-a55d-49eb-af74-cc19e895356f';

-- This removes any potentially existing effort field from all work items
-- and the work item type definition of the type story.
UPDATE work_items SET fields=(fields - 'effort') WHERE type='6ff83406-caa7-47a9-9200-4ca796be11bb';
UPDATE work_item_types SET fields=(fields - 'effort') WHERE id='6ff83406-caa7-47a9-9200-4ca796be11bb';
