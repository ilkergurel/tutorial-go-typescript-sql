-- Links from stakeholder needs (STK) to system requirements (SYS) to test
-- cases (TC) form a graph: TC-10 verifies two requirements, and a careless
-- link closes a loop (SYS-2 and SYS-3 point at each other).
CREATE TABLE trace_links (
  from_code text NOT NULL,
  to_code   text NOT NULL,
  PRIMARY KEY (from_code, to_code)
);
INSERT INTO trace_links VALUES
  ('STK-1', 'SYS-1'), ('STK-1', 'SYS-2'),
  ('SYS-1', 'TC-10'), ('SYS-2', 'TC-10'), ('SYS-2', 'TC-11'),
  ('SYS-2', 'SYS-3'), ('SYS-3', 'SYS-2');

-- Naive: UNION ALL with no cycle check would never stop on this data.
-- A depth guard stops it, after the loop has repeated rows up to the guard.
WITH RECURSIVE reach AS (
  SELECT 'STK-1'::text AS code, 0 AS hops
  UNION ALL
  SELECT l.to_code, r.hops + 1
  FROM reach r
  JOIN trace_links l ON l.from_code = r.code
  WHERE r.hops < 6
)
SELECT code, count(*) AS times_reached
FROM reach
GROUP BY code
ORDER BY code;

-- CYCLE (PG 14) tracks the path and stops at a node already on it.
WITH RECURSIVE reach AS (
  SELECT 'STK-1'::text AS code, 0 AS hops
  UNION ALL
  SELECT l.to_code, r.hops + 1
  FROM reach r
  JOIN trace_links l ON l.from_code = r.code
) CYCLE code SET is_cycle USING path
SELECT hops, code, is_cycle, path
FROM reach
ORDER BY path;

-- Only need the set of reachable nodes? UNION discards rows it has already
-- produced, so the loop runs dry on its own.
WITH RECURSIVE reach(code) AS (
  SELECT 'STK-1'::text
  UNION
  SELECT l.to_code
  FROM reach r
  JOIN trace_links l ON l.from_code = r.code
)
SELECT code FROM reach ORDER BY code;
