-- This removes any potentially existing "resolution" field from all
-- "impediment" work items and the work item type definition of the
-- "impediment". This is needed because a recent space template change
-- (https://github.com/fabric8-services/fabric8-wit/pull/2133) removed or
-- switched the order of the values in this impediment type and therefore
-- couldn't be applied. When we remove the "resolution" field here it here and
-- then import the agile space template, it will create a new "resolution" field
-- on the "impediment" work item type.
--
-- (for an error description see
-- https://github.com/openshiftio/openshift.io/issues/3879)
UPDATE work_items SET fields=(fields - 'resolution') WHERE type='03b9bb64-4f65-4fa7-b165-494cd4f01401';
UPDATE work_item_types SET fields=(fields - 'resolution') WHERE id='03b9bb64-4f65-4fa7-b165-494cd4f01401';