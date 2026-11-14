-- LEARN(fullstack.tracelite.step-12-search): Turkish lowering on both sides, stems plus words in one tsvector, a folded trigram key for other keyboards.
CREATE TABLE requirements (  -- step 2's table, abridged
  project_id  int NOT NULL,
  code        text COLLATE "C" NOT NULL,
  title       text NOT NULL,
  description text NOT NULL DEFAULT '',
  CONSTRAINT requirements_code_key UNIQUE (project_id, code)
);

-- migrations/005_search.sql
-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE FUNCTION tr_lower(text) RETURNS text  -- I to ı, İ to i, whatever the database's locale
  LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
  RETURN lower($1 COLLATE "tr-x-icu");
CREATE FUNCTION search_key(text) RETURNS text  -- for keyboards without Turkish letters
  LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
  RETURN lower(unaccent('unaccent'::regdictionary, $1));
CREATE FUNCTION words_query(q text) RETURNS tsquery  -- every word as a stem, or every word as a prefix
  LANGUAGE sql STABLE PARALLEL SAFE STRICT
  RETURN plainto_tsquery('turkish', tr_lower(q)) ||
    (SELECT to_tsquery('simple', string_agg(quote_literal(w) || ':*', ' & '))
     FROM regexp_split_to_table(tr_lower(q), '[^[:alnum:]]+') AS w WHERE w <> '');
ALTER TABLE requirements ADD COLUMN search tsvector GENERATED ALWAYS AS (
  setweight(to_tsvector('turkish', tr_lower(title)) || to_tsvector('simple', tr_lower(title)), 'A') ||
  setweight(to_tsvector('turkish', tr_lower(description)) || to_tsvector('simple', tr_lower(description)), 'B')
) STORED;
CREATE INDEX requirements_search ON requirements USING gin (search);
CREATE INDEX requirements_title_key ON requirements USING gin (search_key(title) gin_trgm_ops);
-- end of 005_search.sql

INSERT INTO requirements (project_id, code, title, description) VALUES
  (1, 'AUTH',               'Kimlik doğrulama', ''),
  (1, 'AUTH-LOGIN',         'Parola ile giriş', 'Kullanıcı e-posta adresi ve parolasıyla giriş yapar.'),
  (1, 'AUTH-LOGIN-2FA',     'İki adımlı doğrulama', 'Girişte SMS ile tek kullanımlık kod istenir.'),
  (1, 'AUTH-LOGIN-LOCKOUT', 'Hatalı girişte hesap kilitleme', 'Beş hatalı denemeden sonra hesap kilitlenir.'),
  (1, 'AUTH-RESET',         'Parola sıfırlama', 'Sıfırlama bağlantısı e-postayla gönderilir.'),
  (1, 'UI-DARK',            'KARANLIK TEMA', 'Gece kullanımı için koyu renkler.');

-- What the index holds: stems (turkish) and whole words (simple), title A, description B.
SELECT title, description, search FROM requirements WHERE code = 'UI-DARK';

-- The query side goes through the same tr_lower.
SELECT words_query('Parola sıfırlama') AS turkish_keyboard,
       words_query('PAROLA SIFIRLAMA') AS caps_lock,
       words_query('paro') AS typing;

-- How users type, and which of the three matchers finds what.
SELECT typed,
  (SELECT string_agg(code, ' ' ORDER BY code) FROM requirements
    WHERE code >= upper(typed COLLATE "C") AND code < upper(typed COLLATE "C") || '~') AS by_code,
  (SELECT string_agg(code, ' ' ORDER BY code) FROM requirements
    WHERE search @@ words_query(typed)) AS by_words,
  (SELECT string_agg(code, ' ' ORDER BY code) FROM requirements
    WHERE search_key(typed) <% search_key(title)) AS by_trigram
FROM (VALUES ('auth-l'), ('PAROLA SIFIRLAMA'), ('parolalar'), ('paro'), ('karanlık'),
             ('IKI ADIMLI'), ('sifirlma'), ('kilitlenir'), ('e-posta')) AS t(typed);

-- The caps-lock query again, lowered with the database's rules instead: I becomes i.
SELECT plainto_tsquery('turkish', lower('PAROLA SIFIRLAMA')) AS db_lower,
       plainto_tsquery('turkish', tr_lower('PAROLA SIFIRLAMA')) AS tr_lower;
