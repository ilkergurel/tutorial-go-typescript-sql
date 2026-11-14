CREATE EXTENSION pg_trgm;
CREATE EXTENSION unaccent;
SELECT unaccent('Şanlıurfa, Iğdır, İzmir, Gümüşhane, Çeşme') AS unaccented;

-- One search key for every way of typing a name. RETURN binds unaccent and its
-- dictionary now; IMMUTABLE is a promise we make, not one Postgres checks.
CREATE FUNCTION search_key(text) RETURNS text
  LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
  RETURN lower(unaccent('unaccent'::regdictionary, $1));

CREATE TABLE places (name text PRIMARY KEY);
INSERT INTO places VALUES
  ('Adana'), ('Ankara'), ('Antalya'), ('Aydın'), ('Çanakkale'), ('Eskişehir'),
  ('Iğdır'), ('Isparta'), ('İstanbul'), ('İzmir'), ('Kırşehir'), ('Şanlıurfa');
INSERT INTO places SELECT 'Köy ' || g FROM generate_series(1, 10000) AS g;
CREATE INDEX places_name_key ON places USING gin (search_key(name) gin_trgm_ops);
ANALYZE places;

SELECT q AS typed, search_key(q) AS key, best.name AS nearest,
       round(similarity(search_key(best.name), search_key(q))::numeric, 2) AS sim
FROM (VALUES ('Istambul'), ('IZMIR'), ('eskisehir'), ('sanliurfa'),
             ('igdir'), ('ızmır')) AS t(q),
     LATERAL (SELECT name FROM places
              ORDER BY search_key(name) <-> search_key(q), name LIMIT 1) AS best
ORDER BY q;

-- The index serves queries that spell the same expression.
EXPLAIN (COSTS OFF)
SELECT name FROM places WHERE search_key(name) % search_key('SANLIURFA');

-- The price of folding: distinct Turkish words get the same key.
SELECT search_key('kır') = search_key('kir') AS "kır = kir",
       search_key('ılık') = search_key('ilik') AS "ılık = ilik";

-- unaccent itself is only STABLE, so it can't go into an index directly.
CREATE INDEX ON places USING gin (lower(unaccent(name)) gin_trgm_ops);
