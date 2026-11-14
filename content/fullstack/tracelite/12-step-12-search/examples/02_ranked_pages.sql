-- Example 1's table, migration 005 and words_query, condensed.
CREATE EXTENSION pg_trgm;
CREATE EXTENSION unaccent;
CREATE FUNCTION tr_lower(text) RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
  RETURN lower($1 COLLATE "tr-x-icu");
CREATE FUNCTION search_key(text) RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
  RETURN lower(unaccent('unaccent'::regdictionary, $1));
CREATE FUNCTION words_query(q text) RETURNS tsquery LANGUAGE sql STABLE PARALLEL SAFE STRICT
  RETURN plainto_tsquery('turkish', tr_lower(q)) ||
    (SELECT to_tsquery('simple', string_agg(quote_literal(w) || ':*', ' & '))
     FROM regexp_split_to_table(tr_lower(q), '[^[:alnum:]]+') AS w WHERE w <> '');
CREATE TABLE requirements (
  project_id int NOT NULL, code text COLLATE "C" NOT NULL, title text NOT NULL,
  description text NOT NULL DEFAULT '',
  search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('turkish', tr_lower(title)) || to_tsvector('simple', tr_lower(title)), 'A') ||
    setweight(to_tsvector('turkish', tr_lower(description)) || to_tsvector('simple', tr_lower(description)), 'B')
  ) STORED,
  CONSTRAINT requirements_code_key UNIQUE (project_id, code));
CREATE INDEX requirements_search ON requirements USING gin (search);
CREATE INDEX requirements_title_key ON requirements USING gin (search_key(title) gin_trgm_ops);

INSERT INTO requirements (project_id, code, title, description) VALUES
  (1, 'AUTH-LOGIN',         'Parola ile giriş', 'Kullanıcı e-posta adresi ve parolasıyla giriş yapar.'),
  (1, 'AUTH-LOGIN-LOCKOUT', 'Hatalı girişte hesap kilitleme', 'Beş hatalı parola denemesinden sonra hesap kilitlenir.'),
  (1, 'AUTH-RESET',         'Parola sıfırlama', 'Sıfırlama bağlantısı e-postayla gönderilir.'),
  (1, 'AUTH-RESET-EXPIRY',  'Sıfırlama bağlantısının süresi', 'Bağlantı bir saat sonra geçersizdir; yeni parola istenir.'),
  (1, 'PAROLA-KURAL',       'Parola kuralları', 'En az on iki karakter.'),
  (1, 'PROFILE',            'Profil sayfası', 'Kullanıcı adını ve parolasını değiştirebilir.');

-- queries/search.sql, SearchRequirements. $1 project, $2 what the user typed,
-- $3-$5 the last row's sort key (tier 0 on the first page), $6 page size + 1.
PREPARE search(int, text, int, int, text, int) AS
SELECT tier, score, code, title
FROM (
  SELECT r.code, r.title,
    CASE WHEN r.code >= q.prefix AND r.code < q.prefix || '~' THEN 1
         WHEN r.search @@ q.words THEN 2
         ELSE 3 END AS tier,
    CASE WHEN r.code >= q.prefix AND r.code < q.prefix || '~' THEN 0
         WHEN r.search @@ q.words THEN (1e6 * ts_rank(r.search, q.words))::int
         ELSE (1e6 * word_similarity(q.key, search_key(r.title)))::int END AS score
  FROM (SELECT upper($2 COLLATE "C") AS prefix, words_query($2) AS words, search_key($2) AS key) AS q,
       requirements AS r
  WHERE r.project_id = $1
    AND (r.code >= q.prefix AND r.code < q.prefix || '~'
         OR r.search @@ q.words
         OR q.key <% search_key(r.title))
) AS hits
WHERE (tier, -score, code) > ($3, -$4, $5)
ORDER BY tier, score DESC, code
LIMIT $6;

-- Page 1 of "parola", three per page: the API asks for four.
EXECUTE search(1, 'parola', 0, 0, '', 4);
-- Page 2: after the third row's key, which nextCursor carried as [2, 607927, "AUTH-RESET"].
EXECUTE search(1, 'parola', 2, 607927, 'AUTH-RESET', 4);
-- Typed with a typo, and typed on an English keyboard.
EXECUTE search(1, 'sifirlma baglantisi', 0, 0, '', 4);
