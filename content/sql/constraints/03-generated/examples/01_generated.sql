-- LEARN(sql.constraints.generated): derived values the application cannot get wrong.
CREATE TABLE products (
  id          int  PRIMARY KEY,
  name        text NOT NULL,
  net_cents   int  NOT NULL,
  vat_pct     int  NOT NULL DEFAULT 20,
  -- No keyword: VIRTUAL in PG 18. Computed on read, takes no space on disk.
  gross_cents int GENERATED ALWAYS AS (net_cents * (100 + vat_pct) / 100),
  -- STORED: computed on INSERT/UPDATE and written like a normal column.
  name_key    text GENERATED ALWAYS AS (lower(name)) STORED
);

INSERT INTO products (id, name, net_cents) VALUES
  (1, 'Iğdır kayısısı', 4500),
  (2, 'İzmir inciri', 12000);
UPDATE products SET vat_pct = 10 WHERE id = 2;   -- gross_cents follows

SELECT id, name, name_key, net_cents, vat_pct, gross_cents
FROM products
ORDER BY id;

-- attgenerated: 's' = stored, 'v' = virtual.
SELECT attname, attgenerated
FROM pg_attribute
WHERE attrelid = 'products'::regclass AND attgenerated <> ''
ORDER BY attnum;

CREATE INDEX ON products (name_key);      -- stored: indexable
CREATE INDEX ON products (gross_cents);   -- virtual: not in PG 18
