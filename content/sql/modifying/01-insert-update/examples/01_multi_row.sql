-- LEARN(sql.modifying.insert-update): one statement per set of rows, not one per row.
CREATE TABLE products (
  id     int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  sku    text NOT NULL UNIQUE,
  name   text NOT NULL,
  price  numeric(10,2) NOT NULL CHECK (price > 0),
  active boolean NOT NULL DEFAULT true
);

-- Multi-row VALUES: one statement, one round trip, one transaction.
-- Always list the columns: the table's column order is not your API.
INSERT INTO products (sku, name, price) VALUES
  ('TEA-01', 'Rize çayı 1 kg',     210.00),
  ('COF-01', 'Türk kahvesi 250 g', 145.00),
  ('LOK-01', 'Lokum 500 g',        180.00);

-- DEFAULT is a value you can write, per row.
INSERT INTO products (sku, name, price, active) VALUES
  ('HEL-01', 'Tahin helvası', 120.00, DEFAULT),
  ('KUR-01', 'Kuruyemiş',     260.00, false);

-- How Go or TS code sends N rows without building SQL text: one array
-- per column (literals here; $1, $2, $3 from pgx), zipped by unnest.
INSERT INTO products (sku, name, price)
SELECT *
FROM unnest(ARRAY['SIM-01', 'POG-01'],
            ARRAY['Simit', 'Poğaça'],
            ARRAY[15.00, 25.00]::numeric[]);

-- The identity column drew one value per inserted row from its sequence.
SELECT id, sku, name, price, active FROM products ORDER BY id;
