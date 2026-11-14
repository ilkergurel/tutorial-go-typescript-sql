-- NOT in the app: the lessons table and search column that docs/ARCHITECTURE.md drew.
CREATE TABLE lessons (
    id      text PRIMARY KEY,
    title   text NOT NULL,
    body_md text NOT NULL,
    search  tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('turkish', title), 'A') ||
        setweight(to_tsvector('turkish', body_md), 'B')) STORED
);
CREATE INDEX lessons_search_idx ON lessons USING gin (search);

INSERT INTO lessons (id, title, body_md) VALUES
  ('sql.transactions.isolation-levels', 'Yalıtım düzeyleri', 'Okuma anomalileri ve kilitler'),
  ('sql.transactions.row-locks', 'Satır kilitleri', 'FOR UPDATE ile satırları kilitlemek'),
  ('go.basics.defer', 'defer ve temizlik', 'Kaynakları kapatmak');

-- What the index holds: stemmed, lowercased lexemes, with position and weight.
SELECT search FROM lessons WHERE id = 'sql.transactions.row-locks';

-- A Ctrl+K query: stemmed the same way, title matches (A) ranked above body (B).
SELECT id, round(ts_rank(search, q)::numeric, 3) AS rank
FROM lessons, websearch_to_tsquery('turkish', 'kilit') AS q
WHERE search @@ q
ORDER BY rank DESC, id;

-- The Turkish trap: this database lowercases I to i, so caps miss the dotless ı.
SELECT q.query, websearch_to_tsquery('turkish', q.query) AS tsquery, count(l.id) AS hits
FROM unnest(ARRAY['yalıtım', 'Yalıtım', 'YALITIM']) WITH ORDINALITY AS q(query, n)
LEFT JOIN lessons l ON l.search @@ websearch_to_tsquery('turkish', q.query)
GROUP BY q.query, q.n
ORDER BY q.n;

-- PostgreSQL 18 makes generated columns VIRTUAL by default, and those can't be indexed.
ALTER TABLE lessons ADD COLUMN search_v tsvector GENERATED ALWAYS AS (to_tsvector('simple', title));
CREATE INDEX ON lessons USING gin (search_v);
