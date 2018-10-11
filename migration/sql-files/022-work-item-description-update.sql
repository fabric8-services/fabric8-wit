-- migrate work items description by replacing 'plain' with 'PlainText' in the 'markup' element of 'system_description' 
update work_items set fields=jsonb_set(fields, '{system_description, markup}', 
  to_jsonb('PlainText'::text)) where fields->>'system_description' is not null;

