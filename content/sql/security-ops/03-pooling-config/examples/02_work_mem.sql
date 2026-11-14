CREATE TABLE trace_links (
  id             int PRIMARY KEY,
  requirement_id int NOT NULL,
  test_case_id   int NOT NULL,
  note           text NOT NULL
);
INSERT INTO trace_links
SELECT g, g % 5000, g % 997, md5(g::text) FROM generate_series(1, 50000) AS g;
ANALYZE trace_links;

-- How the top Sort node ran, read from EXPLAIN ANALYZE's JSON output.
-- The query really runs; its rows are discarded.
CREATE FUNCTION sort_run(query text, OUT method text, OUT stored_in text)
LANGUAGE plpgsql AS $$
DECLARE plan json;
BEGIN
  EXECUTE 'EXPLAIN (ANALYZE, FORMAT JSON) ' || query INTO plan;
  method    := plan -> 0 -> 'Plan' ->> 'Sort Method';
  stored_in := plan -> 0 -> 'Plan' ->> 'Sort Space Type';
END $$;

-- Sorting all 50,000 rows in memory takes about 4.6 MB.
SET LOCAL work_mem = '4MB';  -- the built-in default
SELECT * FROM sort_run('SELECT * FROM trace_links ORDER BY note');
SET LOCAL work_mem = '8MB';
SELECT * FROM sort_run('SELECT * FROM trace_links ORDER BY note');

-- Back to 4MB, but only the first 20 rows are wanted.
SET LOCAL work_mem = '4MB';
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT * FROM trace_links ORDER BY note LIMIT 20;
