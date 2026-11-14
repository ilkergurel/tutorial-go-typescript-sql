CREATE TABLE customers (
  id         int PRIMARY KEY,
  name       text,
  email      text,
  deleted_at timestamptz
);
CREATE TABLE orders (
  id          int PRIMARY KEY,
  customer_id int NOT NULL REFERENCES customers ON DELETE CASCADE,
  total       numeric(10,2) NOT NULL
);
INSERT INTO customers VALUES
  (1, 'Ayşe Kaya', 'ayse@example.com', NULL),
  (2, 'Can Demir', 'can@example.com',  NULL);
INSERT INTO orders VALUES (10, 1, 250.00), (11, 2, 720.00), (12, 2, 35.00);

-- Soft-delete Can. The foreign key only knows DELETE: CASCADE doesn't
-- fire, and a new order for him still passes the FK check.
UPDATE customers SET deleted_at = '2026-11-01 10:00+03' WHERE id = 2;
INSERT INTO orders VALUES (13, 2, 99.00);

-- Every join to customers now needs its own "AND c.deleted_at IS NULL".
SELECT c.name, count(*) AS orders, sum(o.total) AS revenue
FROM orders o
JOIN customers c ON c.id = o.customer_id
GROUP BY c.name
ORDER BY c.name;

-- If the orders must outlive the customer (invoices, tax records),
-- keep the row but erase what identifies him: the FK stays valid.
UPDATE customers SET name = NULL, email = NULL WHERE id = 2;
SELECT id, name, email, deleted_at IS NOT NULL AS deleted
FROM customers ORDER BY id;

-- If they needn't, a real DELETE lets the declared CASCADE do its job.
DELETE FROM customers WHERE id = 2;
SELECT id, customer_id, total FROM orders ORDER BY id;
