-- LEARN(sql.basics.set-based): describe the result; the planner writes the loop.
CREATE TABLE orders (
  id       int PRIMARY KEY,
  customer text NOT NULL,
  total    numeric(10,2) NOT NULL,
  placed   date NOT NULL
);
INSERT INTO orders VALUES
  (1, 'ayşe',   250.00, '2026-11-02'),
  (2, 'mehmet',  90.00, '2026-11-03'),
  (3, 'ayşe',   400.00, '2026-11-09'),
  (4, 'can',    720.00, '2026-11-10'),
  (5, 'mehmet', 610.00, '2026-10-30'),
  (6, 'can',     35.00, '2026-11-12');

-- "Who spent at least 500 TL in November, biggest spender first?"
-- No loop, no map, no sort call: just the shape of the answer.
SELECT customer, sum(total) AS spent
FROM orders
WHERE placed >= '2026-11-01' AND placed < '2026-12-01'
GROUP BY customer
HAVING sum(total) >= 500
ORDER BY spent DESC;

-- How the server chose to compute it. Read it bottom-up.
EXPLAIN (COSTS OFF)
SELECT customer, sum(total) AS spent
FROM orders
WHERE placed >= '2026-11-01' AND placed < '2026-12-01'
GROUP BY customer
HAVING sum(total) >= 500
ORDER BY spent DESC;
