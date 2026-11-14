CREATE TABLE test_cases (
  id          int  PRIMARY KEY,
  suite       text NOT NULL,
  title       text NOT NULL,
  duration_ms int  NOT NULL
);
INSERT INTO test_cases VALUES
  (1, 'auth',   'İki adımlı doğrulama', 500),
  (2, 'auth',   'Giriş',                300),
  (3, 'auth',   'Çıkış',                300),
  (4, 'auth',   'Şifre sıfırlama',      200),
  (5, 'export', 'CSV dışa aktarım',     900),
  (6, 'export', 'PDF rapor',            400),
  (7, 'export', 'Excel',                100),
  (8, 'search', 'Arama',                 50);

-- The two slowest tests per suite: number each partition, keep 1 and 2.
SELECT suite, title, duration_ms
FROM (
  SELECT suite, title, duration_ms,
         row_number() OVER (PARTITION BY suite ORDER BY duration_ms DESC, id) AS rn
  FROM test_cases
) AS ranked
WHERE rn <= 2
ORDER BY suite, rn;

-- rank() keeps ties: auth's 2nd place is shared, so auth returns three rows.
SELECT suite, title, duration_ms, pos
FROM (
  SELECT suite, title, duration_ms,
         rank() OVER (PARTITION BY suite ORDER BY duration_ms DESC) AS pos
  FROM test_cases
) AS ranked
WHERE pos <= 2
ORDER BY suite, pos, title COLLATE "tr-x-icu";

-- The planner pushes rn <= 2 into the WindowAgg as a run condition.
ANALYZE test_cases;
EXPLAIN (COSTS OFF)
SELECT suite, title
FROM (
  SELECT suite, title,
         row_number() OVER (PARTITION BY suite ORDER BY duration_ms DESC, id) AS rn
  FROM test_cases
) AS ranked
WHERE rn <= 2;

-- An output alias doesn't exist yet in WHERE of the same query level.
SELECT suite, title,
       row_number() OVER (PARTITION BY suite ORDER BY duration_ms DESC, id) AS rn
FROM test_cases
WHERE rn <= 2;
