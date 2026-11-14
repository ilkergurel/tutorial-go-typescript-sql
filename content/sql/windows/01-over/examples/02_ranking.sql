CREATE TABLE test_cases (
  id          int  PRIMARY KEY,
  suite       text NOT NULL,
  title       text NOT NULL,
  duration_ms int  NOT NULL
);
INSERT INTO test_cases VALUES
  (1, 'auth',   'Oturum süresi',        300),
  (2, 'auth',   'Giriş',                300),
  (3, 'auth',   'İki adımlı doğrulama', 500),
  (4, 'auth',   'Çıkış',                300),
  (5, 'auth',   'Şifre sıfırlama',      200),
  (6, 'export', 'CSV dışa aktarım',     900),
  (7, 'export', 'PDF rapor',            900);

-- Slowest first in each suite. Three tests tie at 300 ms: rank and
-- dense_rank give them one number, row_number needs a tie-breaker.
SELECT suite, title, duration_ms,
       rank()       OVER by_time AS rank,
       dense_rank() OVER by_time AS dense_rank,
       row_number() OVER (per_suite ORDER BY duration_ms DESC,
                          title COLLATE "tr-x-icu") AS row_number
FROM test_cases
WINDOW per_suite AS (PARTITION BY suite),
       by_time   AS (per_suite ORDER BY duration_ms DESC)
ORDER BY suite, row_number;

-- Windows run after GROUP BY, so they can rank the groups themselves.
SELECT suite, sum(duration_ms) AS suite_ms,
       rank() OVER (ORDER BY sum(duration_ms) DESC) AS slowest
FROM test_cases
GROUP BY suite
ORDER BY slowest, suite;
