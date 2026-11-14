-- LEARN(sql.search.trigram): fuzzy matching by shared three-letter pieces.
CREATE EXTENSION pg_trgm;
SELECT show_trgm('Antalya') AS antalya, show_trgm('ığdır') AS igdir;

CREATE TABLE cities (name text PRIMARY KEY);
INSERT INTO cities VALUES
  ('Adana'), ('Ankara'), ('Antalya'), ('Aydın'), ('Çanakkale'), ('Eskişehir'),
  ('Iğdır'), ('Isparta'), ('İstanbul'), ('İzmir'), ('Kırşehir'), ('Şanlıurfa');

-- A typo still shares most trigrams. % means: similarity above 0.3.
SELECT name, round(similarity(name, 'Antlya')::numeric, 2) AS sim
FROM cities
WHERE name % 'Antlya';

-- "Did you mean": the nearest name by distance (<-> is 1 - similarity).
SELECT q AS typed, best.name AS nearest,
       round(similarity(best.name, q)::numeric, 2) AS sim
FROM (VALUES ('Istambul'), ('IZMIR'), ('eskisehir'), ('sanliurfa'),
             ('igdir'), ('ızmır')) AS t(q),
     LATERAL (SELECT name FROM cities ORDER BY name <-> q, name LIMIT 1) AS best
ORDER BY q;
