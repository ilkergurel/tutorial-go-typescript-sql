-- amount is spread over the table in no particular order, like most columns.
CREATE TABLE payments (
  id     bigint PRIMARY KEY,
  amount int NOT NULL,
  note   text NOT NULL
);
INSERT INTO payments
SELECT g, (g * 7919) % 1000, 'payment ' || g FROM generate_series(1, 20000) AS g;
CREATE INDEX payments_amount ON payments (amount);
ANALYZE payments;

-- 1% of the rows: the index pays off.
EXPLAIN (COSTS OFF)
SELECT max(note) FROM payments WHERE amount < 10;

-- 90% of the rows: the index exists, and the planner ignores it.
EXPLAIN (COSTS OFF)
SELECT max(note) FROM payments WHERE amount < 900;

-- Measure the planner's choice...
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT max(note) FROM payments WHERE amount < 900;

-- ...against the index it rejected. SET LOCAL lasts until the transaction ends.
SET LOCAL enable_seqscan = off;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT max(note) FROM payments WHERE amount < 900;

SET LOCAL enable_bitmapscan = off;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT max(note) FROM payments WHERE amount < 900;
