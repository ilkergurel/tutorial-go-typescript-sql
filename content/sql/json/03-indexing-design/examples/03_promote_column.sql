CREATE TABLE customers (id int PRIMARY KEY, name text NOT NULL);
INSERT INTO customers VALUES (1, 'Acme Yazılım'), (2, 'Boğaziçi Lojistik');

-- Orders arrive as JSON from a shop integration and are kept as received.
CREATE TABLE orders (
  id  int   PRIMARY KEY,
  doc jsonb NOT NULL
);
INSERT INTO orders VALUES
  (1, '{"customer_id": 1, "total": 1250.00, "gift_note": "Kolay gelsin"}'),
  (2, '{"customer_id": 2, "total": 90.50}'),
  (3, '{"customer_id": 1, "total": 410.00, "coupon": "KASIM10"}');

-- Every order has a customer, every report joins and filters on it: promote
-- it. A STORED generated column is typed, constrained and indexable.
ALTER TABLE orders
  ADD COLUMN customer_id int
    GENERATED ALWAYS AS ((doc ->> 'customer_id')::int) STORED
    NOT NULL REFERENCES customers;
CREATE INDEX orders_customer ON orders (customer_id);

SELECT c.name, count(*) AS orders, sum((o.doc ->> 'total')::numeric) AS total
FROM orders AS o
JOIN customers AS c ON c.id = o.customer_id
GROUP BY c.name
ORDER BY c.name;

-- The optional keys stay in the document, where they belong.
SELECT id, doc - 'customer_id' - 'total' AS extras
FROM orders
ORDER BY id;

-- The constraint now runs on write: an order for an unknown customer fails
-- at INSERT time, not in next month's report.
INSERT INTO orders VALUES (4, '{"customer_id": 99, "total": 10}');
