CREATE INDEX work_items_fields_index on work_items USING gin(fields);