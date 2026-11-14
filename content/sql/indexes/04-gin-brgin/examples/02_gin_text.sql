-- Full-text search: the turkish configuration reduces words to stems.
SELECT to_tsvector('turkish', 'Kitapçılar yeni kitapları tanıttı') AS document,
       to_tsquery('turkish', 'kitaplar') AS query;

CREATE TABLE articles (id int PRIMARY KEY, title text NOT NULL, body text NOT NULL);
INSERT INTO articles
SELECT g,
       (ARRAY['İstanbul', 'Ankara', 'İzmir', 'Bursa', 'Antalya'])[g % 5 + 1] || ' haberleri ' || g,
       (ARRAY['Kitapçılar yeni kitapları tanıttı', 'Belediye otobüs saatlerini değiştirdi',
              'Sahilde bisiklet yolu açıldı', 'Müzede yeni sergi başladı',
              'Pazarda sebze fiyatları düştü'])[g % 5 + 1]
         || ' ' || (ARRAY['bugün', 'dün', 'bu hafta', 'geçen ay'])[g % 4 + 1]
FROM generate_series(1, 20000) AS g;

-- A GIN index on the tsvector: for every stem, the rows that contain it.
CREATE INDEX articles_body_fts ON articles USING gin (to_tsvector('turkish', body));
ANALYZE articles;
EXPLAIN (COSTS OFF)
SELECT count(*) FROM articles
WHERE to_tsvector('turkish', body) @@ to_tsquery('turkish', 'kitap & dün');
SELECT count(*) FROM articles
WHERE to_tsvector('turkish', body) @@ to_tsquery('turkish', 'kitap & dün');

-- Without the configuration argument it's a different expression.
EXPLAIN (COSTS OFF)
SELECT count(*) FROM articles
WHERE to_tsvector(body) @@ to_tsquery('turkish', 'kitap & dün');

-- Substrings (LIKE '%...%') need pg_trgm: three-character pieces of each word.
CREATE EXTENSION pg_trgm;
SELECT show_trgm('haber');
EXPLAIN (COSTS OFF)
SELECT count(*) FROM articles WHERE title ILIKE '%zmir haberleri 12%';
CREATE INDEX articles_title_trgm ON articles USING gin (title gin_trgm_ops);
EXPLAIN (COSTS OFF)
SELECT count(*) FROM articles WHERE title ILIKE '%zmir haberleri 12%';
SELECT count(*) FROM articles WHERE title ILIKE '%zmir haberleri 12%';

-- Turkish letters: what counts as a letter comes from the database's LC_CTYPE.
SELECT datctype = 'C' AS ctype_is_c,
       show_trgm('ığdır') AS igdir,
       similarity('ığdır', 'dür') AS vs_dur,
       similarity('IĞDIR', 'ığdır') AS upper_case
FROM pg_database WHERE datname = current_database();
