-- See https://github.com/almighty/almighty-core/issues/518 for an explanation
-- why these triggers were problematic.
DROP TRIGGER update_WI_after_WILT_trigger ON work_item_types;
DROP FUNCTION update_WI_after_WIT();