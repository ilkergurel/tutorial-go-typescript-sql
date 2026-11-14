CREATE TABLE customers (id int PRIMARY KEY, city text NOT NULL);
CREATE TABLE orders (
  id          int PRIMARY KEY,
  customer_id int NOT NULL REFERENCES customers,
  total       int NOT NULL
);
INSERT INTO customers
SELECT g, (ARRAY['İstanbul', 'Ankara', 'İzmir', 'Bursa'])[g % 4 + 1]
FROM generate_series(1, 2000) AS g;
INSERT INTO orders SELECT g, g % 2000 + 1, g % 300 FROM generate_series(1, 20000) AS g;
CREATE INDEX orders_customer ON orders (customer_id);
ANALYZE customers, orders;

-- İzmir's 500 customers and their 5,000 orders. The planner's choice:
EXPLAIN (COSTS OFF)
SELECT sum(o.total) FROM customers c JOIN orders o ON o.customer_id = c.id
WHERE c.city = 'İzmir';

-- Its price in pages...
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT sum(o.total) FROM customers c JOIN orders o ON o.customer_id = c.id
WHERE c.city = 'İzmir';

-- ...against the alternatives it rejected, one switched off at a time.
SET LOCAL enable_hashjoin = off;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT sum(o.total) FROM customers c JOIN orders o ON o.customer_id = c.id
WHERE c.city = 'İzmir';

SET LOCAL enable_nestloop = off;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT sum(o.total) FROM customers c JOIN orders o ON o.customer_id = c.id
WHERE c.city = 'İzmir';

-- Nothing left to choose: "off" discourages, it doesn't forbid.
SET LOCAL enable_mergejoin = off;
EXPLAIN (COSTS OFF)
SELECT sum(o.total) FROM customers c JOIN orders o ON o.customer_id = c.id
WHERE c.city = 'İzmir';
