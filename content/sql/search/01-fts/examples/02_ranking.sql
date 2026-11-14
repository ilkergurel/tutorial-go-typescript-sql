CREATE TABLE requirements (
  code  text PRIMARY KEY,
  title text NOT NULL,
  body  text NOT NULL
);
INSERT INTO requirements VALUES
  ('REQ-1', 'Audit log export', 'Administrators download the full log as CSV.'),
  ('REQ-2', 'Signed reports', 'Each report export is signed and noted in the audit log.'),
  ('REQ-3', 'Data retention', 'An export runs nightly. Records older than a year are '
            'deleted, and every deletion is recorded in the audit trail.'),
  ('REQ-4', 'Password reset', 'Reset links expire after an hour; the audit log keeps each request.');

-- The same words, ranked three ways. Ranks are floats: round them for display.
SELECT code,
       round(ts_rank(flat, q)::numeric, 4)    AS rank_flat,
       round(ts_rank(doc, q)::numeric, 4)     AS rank_weighted,
       round(ts_rank_cd(doc, q)::numeric, 4)  AS rank_cd
FROM requirements,
     websearch_to_tsquery('english', 'audit export') AS q,
     LATERAL (SELECT to_tsvector('english', title || ' ' || body) AS flat,
                     setweight(to_tsvector('english', title), 'A') ||
                     setweight(to_tsvector('english', body), 'B') AS doc) AS v
WHERE doc @@ q
ORDER BY rank_weighted DESC, code;

-- ts_headline re-reads the original text and marks what matched.
SELECT code,
       ts_headline('english', body, websearch_to_tsquery('english', 'audit export'),
                   'StartSel=[, StopSel=], MaxFragments=2, MaxWords=6, MinWords=3')
         AS snippet
FROM requirements
WHERE code IN ('REQ-2', 'REQ-3')
ORDER BY code;
