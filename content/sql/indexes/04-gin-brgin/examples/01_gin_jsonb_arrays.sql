-- LEARN(sql.indexes.gin-brgin): GIN indexes the parts of a value: keys, elements, words.
CREATE TABLE products (
  id    int PRIMARY KEY,
  attrs jsonb NOT NULL,
  tags  text[] NOT NULL
);
INSERT INTO products
SELECT g,
       jsonb_build_object(
         'brand', 'marka-' || g % 200,
         'color', (ARRAY['kırmızı', 'mavi', 'yeşil', 'siyah'])[g % 4 + 1],
         'stock', g % 50),
       ARRAY[(ARRAY['indirim', 'yeni', 'outlet', 'premium'])[g % 4 + 1], 'kategori-' || g % 100]
FROM generate_series(1, 20000) AS g;
ANALYZE products;

-- jsonb_ops (the default): an entry for every key and every value.
CREATE INDEX products_attrs ON products USING gin (attrs);
EXPLAIN (COSTS OFF)
SELECT id FROM products WHERE attrs @> '{"brand": "marka-7", "color": "siyah"}';

-- Key existence is a GIN operator too...
EXPLAIN (COSTS OFF)
SELECT id FROM products WHERE attrs ? 'discount';

-- ...but ->> yields text, and = on text is not a GIN operator.
EXPLAIN (COSTS OFF)
SELECT id FROM products WHERE attrs ->> 'brand' = 'marka-7';

-- jsonb_path_ops: containment only, in a smaller index.
CREATE INDEX products_attrs_path ON products USING gin (attrs jsonb_path_ops);
SELECT pg_relation_size('products_attrs') / 8192 AS jsonb_ops_pages,
       pg_relation_size('products_attrs_path') / 8192 AS jsonb_path_ops_pages;

-- Arrays: contains (@>) and overlaps (&&) use the index; = ANY doesn't.
CREATE INDEX products_tags ON products USING gin (tags);
EXPLAIN (COSTS OFF)
SELECT id FROM products WHERE tags @> ARRAY['kategori-7', 'yeni'];
EXPLAIN (COSTS OFF)
SELECT id FROM products WHERE 'kategori-7' = ANY (tags);
