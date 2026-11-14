-- Diffing two lists is set algebra, not nested loops or a lookup map.
CREATE TABLE stock (sku text PRIMARY KEY);
CREATE TABLE supplier_file (sku text);
INSERT INTO stock VALUES ('A-1'), ('A-2'), ('B-7'), ('C-3');
INSERT INTO supplier_file VALUES ('A-2'), ('B-7'), ('B-7'), ('D-9'), ('E-1');

-- In the file, not in stock: new products.
SELECT sku FROM supplier_file
EXCEPT
SELECT sku FROM stock
ORDER BY sku;

-- In stock, not in the file: discontinued?
SELECT sku FROM stock
EXCEPT
SELECT sku FROM supplier_file
ORDER BY sku;

-- In both. B-7 appears twice in the file but once here: set semantics.
SELECT sku FROM supplier_file
INTERSECT
SELECT sku FROM stock
ORDER BY sku;

-- Everything either side knows of. UNION removes duplicates; UNION ALL
-- would return all 9 rows and skip the work of finding them.
SELECT sku FROM supplier_file
UNION
SELECT sku FROM stock
ORDER BY sku;
