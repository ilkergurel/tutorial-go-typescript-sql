-- LEARN(design.paradigms.generics-absence): NULL belongs to every type; an aggregate over nothing returns it, and no row is not a NULL.
CREATE TABLE products (name text PRIMARY KEY);
CREATE TABLE offers (
  product     text NOT NULL REFERENCES products,
  price_kurus int CHECK (price_kurus >= 0) -- NULL: price on request
);
INSERT INTO products VALUES ('kahve'), ('simit'), ('su');
INSERT INTO offers VALUES
  ('simit', 1500), ('simit', 1250),
  ('su', 500), ('su', 0), ('su', NULL);

SELECT p.name,
       min(o.price_kurus)   AS cheapest,
       count(o.product)     AS offers,
       count(o.price_kurus) AS priced
FROM products p
LEFT JOIN offers o ON o.product = p.name
GROUP BY p.name
ORDER BY p.name;

-- 'ayran' isn't listed: asking for its row returns no row at all...
SELECT name FROM products WHERE name = 'ayran';

-- ...while an aggregate over no rows still returns one row.
SELECT min(price_kurus) AS cheapest, count(*) AS offers
FROM offers WHERE product = 'ayran';

-- Saying what NULL means, where the reader needs words instead of a gap.
SELECT p.name, coalesce(min(o.price_kurus)::text, 'no offers') AS cheapest
FROM products p
LEFT JOIN offers o ON o.product = p.name
GROUP BY p.name
ORDER BY p.name;
