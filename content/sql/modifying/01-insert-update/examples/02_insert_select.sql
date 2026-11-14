CREATE TABLE products (
  sku    text PRIMARY KEY,
  name   text NOT NULL,
  price  numeric(10,2) NOT NULL,
  active boolean NOT NULL DEFAULT true
);
INSERT INTO products (sku, name, price) VALUES ('TEA-01', 'Rize çayı 1 kg', 210.00);

-- A supplier's CSV landed in a staging table as raw text, Turkish
-- number format included (decimal comma, dot for thousands).
CREATE TABLE staging (sku text, name text, price_text text);
INSERT INTO staging VALUES
  ('BAK-01', 'Fıstıklı baklava 1 kg', '1.250,00'),
  ('SIM-01', 'Simit',                 '15,00'),
  ('TEA-01', 'Rize çayı (dup)',       '1,00');

-- INSERT ... SELECT: convert, filter and insert in one statement.
-- New products start inactive; skus we already sell are skipped.
INSERT INTO products (sku, name, price, active)
SELECT s.sku,
       s.name,
       replace(replace(s.price_text, '.', ''), ',', '.')::numeric,
       false
FROM staging s
WHERE NOT EXISTS (SELECT 1 FROM products p WHERE p.sku = s.sku);

SELECT sku, name, price, active FROM products ORDER BY sku;
