-- This removes any potentially existing system.number field from any work item.
UPDATE work_items SET fields=(fields - 'system.number');