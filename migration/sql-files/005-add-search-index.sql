CREATE INDEX IF NOT EXISTS fulltext_search_index ON work_items 
USING gin((setweight(to_tsvector('english',coalesce(fields->>'system.title','')),'B')||
    setweight(to_tsvector('english',coalesce(fields->>'system.description','')),'C')|| 
    setweight(to_tsvector('english', id::text),'A')));