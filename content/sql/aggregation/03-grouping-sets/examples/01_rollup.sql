-- LEARN(sql.aggregation.grouping-sets): several groupings in one pass; GROUPING() tells them apart.
CREATE TABLE test_runs (
  id          int PRIMARY KEY,
  suite       text NOT NULL,
  browser     text,             -- NULL: an API test, no browser involved
  duration_ms int NOT NULL
);
INSERT INTO test_runs VALUES
  (1, 'auth',   'chrome', 120), (2, 'auth', 'firefox', 340),
  (3, 'auth',   NULL,      30), (4, 'auth', NULL,       40),
  (5, 'export', 'chrome', 900), (6, 'search', 'chrome', 80);

-- Detail rows, a subtotal per suite and a grand total, in one query.
-- Look at auth: two rows with browser NULL that mean different things.
SELECT suite, browser, count(*) AS runs, sum(duration_ms) AS total_ms
FROM test_runs
GROUP BY ROLLUP (suite, browser)
ORDER BY suite, browser, GROUPING(browser);

-- GROUPING(col) is 1 where col was rolled up, so a subtotal's NULL can be
-- told apart from a NULL in the data. The two-argument form is a bitmask.
SELECT CASE WHEN GROUPING(suite) = 1 THEN 'all suites' ELSE suite END
         AS suite_label,
       CASE WHEN GROUPING(browser) = 1 THEN 'all browsers'
            ELSE coalesce(browser, 'api') END AS browser_label,
       GROUPING(suite, browser) AS level,
       count(*) AS runs
FROM test_runs
GROUP BY ROLLUP (suite, browser)
ORDER BY GROUPING(suite), suite, GROUPING(browser), browser;
