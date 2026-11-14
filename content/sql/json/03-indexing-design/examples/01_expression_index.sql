-- LEARN(sql.json.indexing-design): index the expression your query writes, not the document.
CREATE TABLE tickets (
  id  int   PRIMARY KEY,
  doc jsonb NOT NULL
);
INSERT INTO tickets
SELECT g, jsonb_build_object(
         'priority', CASE WHEN g % 100 = 0 THEN 1 ELSE 3 + g % 3 END,
         'reporter', 'user-' || g % 300,
         'labels',   CASE WHEN g % 50 = 0 THEN '["kvkk", "ui"]'::jsonb
                          ELSE '["ui"]'::jsonb END)
FROM generate_series(1, 20000) AS g;

CREATE INDEX tickets_doc ON tickets USING gin (doc jsonb_path_ops);
CREATE INDEX tickets_priority ON tickets (((doc ->> 'priority')::int));
ANALYZE tickets;

-- GIN with jsonb_path_ops serves jsonpath matches (@?, @@) as well as @>.
EXPLAIN (COSTS OFF)
SELECT id FROM tickets WHERE doc @? '$.labels[*] ? (@ == "kvkk")';

-- Ranges and sorting on one key are B-tree work: the expression index.
EXPLAIN (COSTS OFF)
SELECT id FROM tickets WHERE (doc ->> 'priority')::int <= 1;
EXPLAIN (COSTS OFF)
SELECT id, doc ->> 'reporter' AS reporter
FROM tickets
ORDER BY (doc ->> 'priority')::int
LIMIT 5;

-- A different expression with the same meaning doesn't match the index.
EXPLAIN (COSTS OFF)
SELECT id FROM tickets WHERE (doc -> 'priority')::int <= 1;
