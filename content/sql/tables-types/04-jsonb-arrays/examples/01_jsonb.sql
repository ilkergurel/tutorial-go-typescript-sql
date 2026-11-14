-- json keeps the text you typed; jsonb parses it: keys reordered,
-- duplicates dropped (the last one wins).
SELECT '{"b": 1, "a": 2, "a": 3}'::json  AS as_json,
       '{"b": 1, "a": 2, "a": 3}'::jsonb AS as_jsonb;

CREATE TABLE products (
  sku   text PRIMARY KEY,
  name  text NOT NULL,
  price numeric(10, 2) NOT NULL,     -- every product has one: a real column
  attrs jsonb NOT NULL DEFAULT '{}'  -- differs per category: jsonb
);

INSERT INTO products VALUES
  ('tea-rize', 'Rize çayı',   249.90, '{"origin": "Rize", "weight_g": 1000}'),
  ('tea-earl', 'Earl Grey',   189.50, '{"origin": "Sri Lanka", "weight_g": 250, "organic": true}'),
  ('usb-c',    'USB-C kablo', 129.00, '{"length_m": 2, "watts": 100}');

-- -> returns jsonb, ->> returns text. @> asks "does attrs contain this
-- document?", ? asks "does it have this key?".
SELECT sku, attrs -> 'weight_g' AS weight_json, attrs ->> 'origin' AS origin
FROM products
WHERE attrs @> '{"origin": "Rize"}' OR attrs ? 'organic'
ORDER BY sku;

-- Nothing checks the shape of a jsonb value, so this insert succeeds...
INSERT INTO products VALUES
  ('tea-green', 'Yeşil çay', 99.00, '{"origin": "Rize", "weight_g": "yarım kilo"}');

-- ...and the mistake surfaces later, in somebody else's report.
SELECT sum((attrs ->> 'weight_g')::int) AS total_tea_grams
FROM products
WHERE sku LIKE 'tea-%';
