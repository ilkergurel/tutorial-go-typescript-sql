-- LEARN(sql.search.turkish): the configuration decides what a word becomes.
SELECT cfg, to_tsvector(cfg, 'Raporlar İstanbul''da hazırlandı ve testleri bitti') AS document
FROM unnest(ARRAY['turkish', 'english', 'simple']::regconfig[]) AS cfg;

CREATE TABLE requirements (
  id     int PRIMARY KEY,
  title  text NOT NULL,
  search tsvector GENERATED ALWAYS AS (to_tsvector('turkish', title)) STORED
);
INSERT INTO requirements (id, title) VALUES
  (1, 'Test raporları PDF olarak indirilebilir'),
  (2, 'Haftalık rapor e-posta ile gönderilir'),
  (3, 'Şifreler en az on iki karakter olmalı');

-- This server's default; initdb chose it from the locale.
SET LOCAL default_text_search_config = 'pg_catalog.english';

-- The column holds turkish stems. Which rows does each query find?
SELECT q AS typed,
       (SELECT string_agg(id::text, ',' ORDER BY id) FROM requirements
         WHERE search @@ plainto_tsquery('turkish', q)) AS turkish,
       (SELECT string_agg(id::text, ',' ORDER BY id) FROM requirements
         WHERE search @@ plainto_tsquery(q)) AS no_config,
       (SELECT string_agg(id::text, ',' ORDER BY id) FROM requirements
         WHERE search @@ plainto_tsquery('simple', q)) AS simple
FROM (VALUES ('rapor'), ('raporlar'), ('raporları'), ('şifre'), ('şifreler')) AS t(q);
