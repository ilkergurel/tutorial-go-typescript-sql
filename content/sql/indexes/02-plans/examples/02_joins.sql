CREATE TABLE customers (id int PRIMARY KEY, city text NOT NULL);
CREATE TABLE orders (
  id          int PRIMARY KEY,
  customer_id int NOT NULL REFERENCES customers,
  total       int NOT NULL
);
INSERT INTO customers
SELECT g, (ARRAY['İstanbul', 'Ankara', 'İzmir', 'Bursa'])[g % 4 + 1]
FROM generate_series(1, 2000) AS g;
-- 10 orders per customer.
INSERT INTO orders SELECT g, g % 2000 + 1, g % 300 FROM generate_series(1, 20000) AS g;
CREATE INDEX orders_customer ON orders (customer_id);
ANALYZE customers, orders;

-- Three customers: for each one, look up its orders in the index.
EXPLAIN (COSTS OFF)
SELECT c.id, o.total
FROM customers c JOIN orders o ON o.customer_id = c.id
WHERE c.id IN (1, 2, 3);

-- Every row on both sides: hash the smaller side, stream the larger past it.
EXPLAIN (COSTS OFF)
SELECT c.city, sum(o.total)
FROM customers c JOIN orders o ON o.customer_id = c.id
GROUP BY c.city;

-- The result must come out sorted by the join key: merge two sorted inputs.
EXPLAIN (COSTS OFF)
SELECT c.id, o.id
FROM customers c JOIN orders o ON o.customer_id = c.id
ORDER BY c.id;
