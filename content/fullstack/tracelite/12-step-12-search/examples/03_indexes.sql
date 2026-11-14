-- Migration 005 on a bigger table: which index serves which matcher?
CREATE EXTENSION pg_trgm;
CREATE EXTENSION unaccent;
CREATE FUNCTION tr_lower(text) RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
  RETURN lower($1 COLLATE "tr-x-icu");
CREATE FUNCTION search_key(text) RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
  RETURN lower(unaccent('unaccent'::regdictionary, $1));
CREATE TABLE requirements (
  project_id int NOT NULL, code text COLLATE "C" NOT NULL, title text NOT NULL,
  search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('turkish', tr_lower(title)) || to_tsvector('simple', tr_lower(title)), 'A')) STORED,
  CONSTRAINT requirements_code_key UNIQUE (project_id, code));
CREATE INDEX requirements_search ON requirements USING gin (search);
CREATE INDEX requirements_title_key ON requirements USING gin (search_key(title) gin_trgm_ops);
INSERT INTO requirements (project_id, code, title)
SELECT g % 4, 'REQ-' || g,
       (ARRAY['Parola', 'Rapor', 'Giriş', 'Hesap', 'Ödeme', 'Fatura', 'Bildirim'])[g % 7 + 1] || ' ' ||
       (ARRAY['kuralı', 'ekranı', 'sıfırlama', 'dışa aktarma', 'onayı'])[g % 5 + 1] || ' ' || g
FROM generate_series(1, 30000) AS g;
ANALYZE requirements;

-- Every plan below is GENERIC_PLAN (PG 16): what a prepared statement from pgx
-- gets once the plan cache stops planning for each value.

-- A code prefix: a range over requirements_code_key, whatever $2 turns out to be.
EXPLAIN (GENERIC_PLAN, COSTS OFF)
SELECT code FROM requirements
WHERE project_id = $1 AND code >= upper($2 COLLATE "C") AND code < upper($2 COLLATE "C") || '~';

-- The same prefix as LIKE: the planner can't see the pattern, so no range.
EXPLAIN (GENERIC_PLAN, COSTS OFF)
SELECT code FROM requirements
WHERE project_id = $1 AND code LIKE upper($2 COLLATE "C") || '%';

-- Words: the GIN index on the stored tsvector.
EXPLAIN (GENERIC_PLAN, COSTS OFF)
SELECT code FROM requirements WHERE search @@ plainto_tsquery('turkish', tr_lower($1));

-- Typos and other keyboards: the trigram GIN index on the folded title.
EXPLAIN (GENERIC_PLAN, COSTS OFF)
SELECT code FROM requirements WHERE search_key($1) <% search_key(title);

-- Ordering by similarity needs a GiST trigram index; GIN can only filter.
EXPLAIN (GENERIC_PLAN, COSTS OFF)
SELECT code FROM requirements ORDER BY search_key($1) <<-> search_key(title) LIMIT 5;
