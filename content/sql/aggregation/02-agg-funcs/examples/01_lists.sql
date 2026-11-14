-- LEARN(sql.aggregation.agg-funcs): an aggregate can build a value, not just a number.
CREATE TABLE test_cases (
  id    int PRIMARY KEY,
  suite text NOT NULL,
  title text NOT NULL,
  owner text              -- NULL: nobody owns it yet
);
INSERT INTO test_cases VALUES
  (1, 'auth',   'Şifre sıfırlama',      'Zeynep'),
  (2, 'auth',   'Çıkış',                'Can'),
  (3, 'auth',   'Giriş',                NULL),
  (4, 'auth',   'İki adımlı doğrulama', 'Can'),
  (5, 'export', 'CSV dışa aktarım',     'Ilgaz');

-- The order of elements goes inside the call. string_agg skips NULLs;
-- array_agg keeps them.
SELECT suite,
       string_agg(owner, ', ' ORDER BY id) AS owners_text,
       array_agg(owner ORDER BY id)        AS owners_array,
       array_agg(DISTINCT owner ORDER BY owner) FILTER (WHERE owner IS NOT NULL)
         AS distinct_owners
FROM test_cases
GROUP BY suite
ORDER BY suite;

-- ORDER BY title compares in the column's collation: C.UTF-8 sorts by code
-- point, so Ç, İ and Ş come after Z. Name the Turkish collation to fix it.
SELECT string_agg(title, ', ' ORDER BY title)                     AS by_code_point,
       string_agg(title, ', ' ORDER BY title COLLATE "tr-x-icu") AS turkish
FROM test_cases
WHERE suite = 'auth';

-- No input rows: array_agg gives NULL, not an empty array.
SELECT array_agg(title ORDER BY title)                 AS raw,
       coalesce(array_agg(title ORDER BY title), '{}') AS or_empty
FROM test_cases
WHERE suite = 'billing';

-- With DISTINCT, the ORDER BY may only use the aggregated expression.
SELECT array_agg(DISTINCT owner ORDER BY id) FROM test_cases;
