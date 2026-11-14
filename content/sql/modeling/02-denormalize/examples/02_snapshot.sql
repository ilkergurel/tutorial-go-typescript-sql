-- Copying a value at the moment of an event records a different fact:
-- what was agreed then. That is not redundancy.
CREATE TABLE products (
  sku     text PRIMARY KEY,
  name    text NOT NULL,
  price   numeric(10, 2) NOT NULL,
  vat_pct int NOT NULL
);
CREATE TABLE order_lines (
  order_id   int,
  sku        text REFERENCES products,
  qty        int NOT NULL CHECK (qty > 0),
  unit_price numeric(10, 2) NOT NULL,       -- the price on the order
  vat_pct    int NOT NULL,                  -- the KDV rate that applied
  -- Derived within the row: the database computes it, so it can't drift.
  net_total  numeric(12, 2) GENERATED ALWAYS AS (qty * unit_price) STORED,
  PRIMARY KEY (order_id, sku)
);
INSERT INTO products VALUES ('usb-c', 'USB-C kablo', 129.00, 18);

-- Order 1: copy price and rate from the product as part of the write.
INSERT INTO order_lines (order_id, sku, qty, unit_price, vat_pct)
SELECT 1, sku, 2, price, vat_pct FROM products WHERE sku = 'usb-c';

-- Later: a new price list, and the general KDV rate goes from 18 to 20.
UPDATE products SET price = 149.00, vat_pct = 20 WHERE sku = 'usb-c';

-- The order as invoiced, next to what a join to today's product says.
SELECT l.order_id, l.qty, l.unit_price, l.vat_pct, l.net_total,
       p.price AS price_today, p.vat_pct AS vat_today,
       l.qty * p.price AS net_if_joined
FROM order_lines l
JOIN products p USING (sku)
ORDER BY l.order_id;
