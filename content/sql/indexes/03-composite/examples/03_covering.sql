-- Runtime sql-nontx: each statement commits on its own, so VACUUM can run
-- and record which table pages are visible to every transaction.
CREATE TABLE orders (
  id          bigint PRIMARY KEY,
  customer_id int NOT NULL,
  total       numeric(10,2) NOT NULL,
  note        text NOT NULL
);
INSERT INTO orders
SELECT g, g % 1000 + 1, g % 500 + 0.99, 'order ' || g FROM generate_series(1, 20000) AS g;
CREATE INDEX orders_customer ON orders (customer_id);
VACUUM (ANALYZE) orders;

-- Customer 42's total: the index finds 20 rows, the table supplies total.
EXPLAIN (COSTS OFF)
SELECT sum(total) FROM orders WHERE customer_id = 42;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT sum(total) FROM orders WHERE customer_id = 42;

-- Carry total in the index as payload: no table visits left.
CREATE INDEX orders_customer_total ON orders (customer_id) INCLUDE (total);
EXPLAIN (COSTS OFF)
SELECT sum(total) FROM orders WHERE customer_id = 42;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT sum(total) FROM orders WHERE customer_id = 42;

-- Payload is not a search key...
EXPLAIN (COSTS OFF)
SELECT count(*) FROM orders WHERE total = 42.99;

-- ...and it isn't free: an index with INCLUDE never deduplicates keys.
SELECT pg_relation_size('orders_customer') / 8192 AS key_only_pages,
       pg_relation_size('orders_customer_total') / 8192 AS covering_pages;

-- UNIQUE with INCLUDE: unique on email alone; name only rides along.
CREATE TABLE users (id int PRIMARY KEY, email text NOT NULL, name text NOT NULL);
CREATE UNIQUE INDEX users_email ON users (email) INCLUDE (name);
INSERT INTO users VALUES (1, 'ayse@example.com', 'Ayşe');
INSERT INTO users VALUES (2, 'ayse@example.com', 'Ayşe Yılmaz');
