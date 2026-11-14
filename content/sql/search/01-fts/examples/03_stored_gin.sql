-- The tsvector is computed on write and kept: STORED is required to index it.
CREATE TABLE requirements (
  id     int PRIMARY KEY,
  title  text NOT NULL,
  body   text NOT NULL,
  search tsvector GENERATED ALWAYS AS (
           setweight(to_tsvector('english', title), 'A') ||
           setweight(to_tsvector('english', body), 'B')) STORED
);
INSERT INTO requirements (id, title, body)
SELECT g,
       (ARRAY['Audit log export', 'Password reset', 'Signed reports',
              'Bulk import', 'Session timeout'])[g % 5 + 1] || ' ' || g,
       (ARRAY['Every change is written to the audit log.',
              'Users can export the results as CSV.',
              'Reset links expire after one hour.',
              'Imports validate every row first.'])[g % 4 + 1]
FROM generate_series(1, 20000) AS g;
SELECT title, body, search FROM requirements WHERE id = 5;
CREATE INDEX requirements_search ON requirements USING gin (search);
ANALYZE requirements;

-- Top 5 by rank: the index finds every match, then each one is ranked.
EXPLAIN (COSTS OFF)
SELECT id, title
FROM requirements, websearch_to_tsquery('english', 'audit export') AS q
WHERE search @@ q
ORDER BY ts_rank(search, q) DESC, id
LIMIT 5;

SELECT count(*) AS matches
FROM requirements
WHERE search @@ websearch_to_tsquery('english', 'audit export');

SELECT id, title, round(ts_rank(search, q)::numeric, 4) AS rank
FROM requirements, websearch_to_tsquery('english', 'audit export') AS q
WHERE search @@ q
ORDER BY ts_rank(search, q) DESC, id
LIMIT 5;
