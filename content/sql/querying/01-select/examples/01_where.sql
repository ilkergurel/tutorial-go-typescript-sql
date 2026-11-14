-- LEARN(sql.querying.select): pick columns, filter rows, and know when each clause runs.
CREATE TABLE products (
  id       int PRIMARY KEY,
  name     text NOT NULL,
  category text NOT NULL,
  price    numeric(10,2) NOT NULL,
  stock    int NOT NULL
);
INSERT INTO products VALUES
  (1, 'Çay bardağı seti', 'kitchen',  189.90, 40),
  (2, 'Cezve',            'kitchen',  349.00,  0),
  (3, 'Kilim yastık',     'home',     520.00, 12),
  (4, 'Bakır tepsi',      'kitchen',  899.00,  3),
  (5, 'Termos',           'outdoor',  649.00, 25),
  (6, 'Hamak',            'outdoor', 1150.00,  0);

-- Name the columns you need; computed columns get an alias.
SELECT name, price, price * stock AS stock_value
FROM products
WHERE category = 'kitchen' AND stock > 0
ORDER BY stock_value DESC;

-- AND binds tighter than OR. This reads as
-- "outdoor (any stock) OR (kitchen AND in stock)": Hamak sneaks in.
SELECT name FROM products
WHERE category = 'outdoor' OR category = 'kitchen' AND stock > 0
ORDER BY id;

-- What was meant. IN is a list of ORs, and it spares you the parentheses.
SELECT name FROM products
WHERE category IN ('outdoor', 'kitchen') AND stock > 0
ORDER BY id;

-- BETWEEN includes both ends: 349.00 and 649.00 both match.
SELECT name, price FROM products
WHERE price BETWEEN 349 AND 649
ORDER BY price;

-- WHERE runs before the SELECT list exists, so its aliases don't either.
SELECT name, price * stock AS stock_value
FROM products
WHERE stock_value > 5000;
