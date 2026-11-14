-- Snowball's turkish stemmer strips suffixes by rule, without a dictionary.
SELECT word, to_tsvector('turkish', word) AS lexeme
FROM unnest(ARRAY['rapor', 'raporlar', 'raporları',
                  'hata', 'hatalar', 'hatası',
                  'gereksinim', 'gereksinimler', 'gereksinimleri']) AS word;

-- Apostrophes split suffixes off proper nouns; ts_debug shows every token's path.
SELECT alias, token, dictionary, lexemes
FROM ts_debug('turkish', 'TraceLite''ın raporu İstanbul''da');

CREATE TABLE requirements (id int PRIMARY KEY, title text NOT NULL);
INSERT INTO requirements VALUES
  (1, 'Giriş ekranında hata mesajı'),
  (2, 'Raporlama modülündeki hatalar'),
  (3, 'Hatay şubesi için yeni rapor');

-- Stems alone miss a form; prefixes on unstemmed words catch it, and more.
SELECT id, title,
       to_tsvector('turkish', title) @@ plainto_tsquery('turkish', 'hata') AS stem,
       to_tsvector('simple', title) @@ to_tsquery('simple', 'hata:*')      AS prefix
FROM requirements
ORDER BY id;
