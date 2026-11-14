-- LEARN(sql.indexes.composite): equality columns first, then the range or sort column.
CREATE TABLE orders (
  id        bigint PRIMARY KEY,
  shop_id   int NOT NULL,
  placed_on date NOT NULL,
  total     numeric(10,2) NOT NULL
);
-- 100 shops, 100 orders a day for 200 days: one order per shop per day.
INSERT INTO orders
SELECT g, g % 100 + 1, date '2026-01-01' + g / 100, g % 500 + 0.99
FROM generate_series(0, 19999) AS g;
ANALYZE orders;

-- The question: shop 7's orders in March and April.
SELECT count(*) FROM orders
WHERE shop_id = 7 AND placed_on BETWEEN '2026-03-01' AND '2026-04-30';

-- Range column first: the scan covers two months of every shop's entries.
CREATE INDEX orders_day_shop ON orders (placed_on, shop_id);
EXPLAIN (COSTS OFF)
SELECT id, total FROM orders
WHERE shop_id = 7 AND placed_on BETWEEN '2026-03-01' AND '2026-04-30';
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT id, total FROM orders
WHERE shop_id = 7 AND placed_on BETWEEN '2026-03-01' AND '2026-04-30';
DROP INDEX orders_day_shop;

-- Equality column first: shop 7's entries are adjacent, already sorted by day.
CREATE INDEX orders_shop_day ON orders (shop_id, placed_on);
EXPLAIN (COSTS OFF)
SELECT id, total FROM orders
WHERE shop_id = 7 AND placed_on BETWEEN '2026-03-01' AND '2026-04-30';
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT id, total FROM orders
WHERE shop_id = 7 AND placed_on BETWEEN '2026-03-01' AND '2026-04-30';

-- The same index serves the leading column alone, in index order...
EXPLAIN (COSTS OFF)
SELECT id, total FROM orders WHERE shop_id = 7 ORDER BY placed_on DESC LIMIT 10;

-- ...but a sort on a column it doesn't hold still needs a Sort.
EXPLAIN (COSTS OFF)
SELECT id, total FROM orders WHERE shop_id = 7 ORDER BY total DESC LIMIT 10;
