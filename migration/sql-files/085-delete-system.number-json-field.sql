-- This removes any potentially existing system_number field from all work items.
UPDATE work_items SET fields=(fields - 'system_number');
