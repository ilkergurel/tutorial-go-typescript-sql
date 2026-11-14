-- LEARN(sql.indexes.btree): EXPLAIN predicts, EXPLAIN ANALYZE runs and measures.
CREATE TABLE orders (
  id          bigint PRIMARY KEY,
  customer_id int NOT NULL,
  total       numeric(10,2) NOT NULL
);
INSERT INTO orders
SELECT g, g % 1000 + 1, g % 500 + 0.99 FROM generate_series(1, 20000) AS g;
ANALYZE orders;

-- No index on customer_id: the only plan is to read every row.
EXPLAIN (COSTS OFF)
SELECT id, total FROM orders WHERE customer_id = 42;

CREATE INDEX orders_customer_id ON orders (customer_id);

-- Same query, new plan. Still a prediction: nothing has run.
EXPLAIN (COSTS OFF)
SELECT id, total FROM orders WHERE customer_id = 42;

-- ANALYZE executes it: real row counts, and the pages it touched.
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT id, total FROM orders WHERE customer_id = 42;

-- Plain EXPLAIN of a DELETE only plans it...
EXPLAIN (COSTS OFF)
DELETE FROM orders WHERE customer_id = 42;
-- ...with ANALYZE it executes: this DELETE deletes.
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
DELETE FROM orders WHERE customer_id = 42;
SELECT count(*) AS orders_left FROM orders WHERE customer_id = 42;
