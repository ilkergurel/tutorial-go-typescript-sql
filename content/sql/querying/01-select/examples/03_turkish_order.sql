CREATE TABLE cities (name text PRIMARY KEY);
INSERT INTO cities VALUES
  ('Zonguldak'), ('İzmir'), ('Ankara'), ('Çorum'),
  ('Iğdır'), ('Ağrı'), ('Şırnak'), ('Isparta');

-- The database's collation (here the builtin C.UTF-8) sorts by Unicode
-- code point: every ASCII letter beats Ç, İ, Ş, and ğ comes after n and s.
SELECT name FROM cities ORDER BY name;

-- Turkish alphabetical order is a per-query (or per-column) choice:
-- A, Ağ, An, Ç, I, İ, Ş, Z.
SELECT name FROM cities ORDER BY name COLLATE "tr-x-icu";

-- Comparisons follow the collation too: "names before J" is a
-- different set of rows under each one.
SELECT name FROM cities WHERE name < 'J' ORDER BY name;
SELECT name FROM cities WHERE name < 'J' COLLATE "tr-x-icu"
ORDER BY name COLLATE "tr-x-icu";
