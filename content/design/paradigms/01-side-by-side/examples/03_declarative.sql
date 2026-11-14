-- LEARN(design.paradigms.side-by-side): SQL states the result and the rules; the planner picks the loops.
CREATE TABLE runs (
  test   text PRIMARY KEY,
  req    text NOT NULL,
  status text NOT NULL CHECK (status IN ('passed', 'failed'))
);
INSERT INTO runs VALUES
  ('TC-1', 'REQ-LOGIN', 'passed'),  ('TC-2', 'REQ-LOGIN', 'failed'),
  ('TC-3', 'REQ-LOGIN', 'passed'),  ('TC-4', 'REQ-EXPORT', 'passed'),
  ('TC-5', 'REQ-SEARCH', 'failed'), ('TC-6', 'REQ-SEARCH', 'failed');

SELECT req,
       count(*) FILTER (WHERE status = 'passed') AS passed,
       count(*) FILTER (WHERE status = 'failed') AS failed
FROM runs
GROUP BY req
ORDER BY failed DESC, req;

-- The map and the sort you wrote by hand in Go, chosen by the planner here.
EXPLAIN (COSTS OFF)
SELECT req, count(*) FILTER (WHERE status = 'failed') AS failed
FROM runs GROUP BY req ORDER BY failed DESC, req;

-- The rule lives in the table, so it holds for every client and every write.
INSERT INTO runs VALUES ('TC-7', 'REQ-LOGIN', 'pased');
