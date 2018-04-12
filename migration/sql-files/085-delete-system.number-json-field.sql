-- This removes any potentially existing system.number field from all work items.
UPDATE work_items SET fields=(fields - 'system.number');
