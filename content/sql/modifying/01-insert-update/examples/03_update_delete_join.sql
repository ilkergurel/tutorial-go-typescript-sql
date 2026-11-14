CREATE TABLE products (sku text PRIMARY KEY, price numeric(10,2) NOT NULL);
INSERT INTO products VALUES
  ('TEA-01', 210.00), ('COF-01', 145.00), ('LOK-01', 180.00),
  ('SIM-01', 15.00),  ('BAK-01', 1250.00);

CREATE TABLE price_list (sku text PRIMARY KEY, new_price numeric(10,2) NOT NULL);
INSERT INTO price_list VALUES ('TEA-01', 235.00), ('LOK-01', 195.00), ('XYZ-99', 1.00);

CREATE TABLE discontinued (sku text PRIMARY KEY);
INSERT INTO discontinued VALUES ('SIM-01'), ('COF-01');

-- UPDATE ... FROM joins the target to other tables; each joined row is
-- one update. Unmatched products (and XYZ-99) are left alone.
UPDATE products p
SET price = l.new_price
FROM price_list l
WHERE l.sku = p.sku;

-- DELETE ... USING: the same join idea, for deletes.
DELETE FROM products p
USING discontinued d
WHERE d.sku = p.sku;

SELECT sku, price FROM products ORDER BY sku;

-- The standard-SQL form, a subquery per row, touches EVERY row: BAK-01
-- has no new price, so it gets NULL. Here NOT NULL stops it.
UPDATE products
SET price = (SELECT l.new_price FROM price_list l WHERE l.sku = products.sku);
