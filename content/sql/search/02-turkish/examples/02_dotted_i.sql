-- to_tsvector lowercases with the database's rules (here C.UTF-8: I -> i, never ı).
SELECT spelling,
       to_tsvector('turkish', spelling)                           AS default_lower,
       to_tsvector('turkish', lower(spelling COLLATE "tr-x-icu")) AS turkish_lower
FROM unnest(ARRAY['IĞDIR', 'Iğdır', 'ığdır', 'ISPARTA', 'ısparta', 'İZMİR', 'IZMIR'])
     AS spelling;

-- A COLLATE on the input doesn't help: only lower() reads the collation.
SELECT to_tsvector('turkish', 'ISPARTA' COLLATE "tr-x-icu") AS collate_ignored;

-- Lowercase the Turkish way on both sides: in the stored column and in the query.
CREATE TABLE news (
  id       int PRIMARY KEY,
  headline text NOT NULL,
  search   tsvector GENERATED ALWAYS AS (
             to_tsvector('turkish', lower(headline COLLATE "tr-x-icu"))) STORED
);
INSERT INTO news (id, headline) VALUES
  (1, 'IĞDIR''DA YENİ HAVALİMANI AÇILDI'),
  (2, 'Iğdır kayısısı Avrupa yolunda'),
  (3, 'Isparta''da gül hasadı başladı');

SELECT q AS typed,
       (SELECT string_agg(id::text, ',' ORDER BY id) FROM news
         WHERE search @@ plainto_tsquery('turkish', q)) AS raw_query,
       (SELECT string_agg(id::text, ',' ORDER BY id) FROM news
         WHERE search @@ plainto_tsquery('turkish', lower(q COLLATE "tr-x-icu")))
         AS turkish_lowered_query
FROM (VALUES ('ığdır'), ('IĞDIR'), ('Iğdır'), ('ISPARTA'), ('igdir')) AS t(q);
