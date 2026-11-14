-- LEARN(sql.json.path-table): a jsonpath walks, filters and unwraps inside one document.
CREATE TABLE ci_reports (
  id     int   PRIMARY KEY,
  report jsonb NOT NULL
);
INSERT INTO ci_reports VALUES
  (1, '{"build": 108, "suites": [
         {"name": "auth", "tests": [
            {"title": "Giriş", "status": "passed", "ms": 120},
            {"title": "Çıkış", "status": "failed", "ms": 85, "error": "timeout"}]},
         {"name": "export", "tests": [
            {"title": "CSV dışa aktarım", "status": "passed", "ms": 900}]}]}'),
  (2, '{"build": 109, "suites": [
         {"name": "auth", "tests": [
            {"title": "Giriş", "status": "passed", "ms": "slow"}]}]}');

-- [*] iterates an array, ? (...) filters, @ is the item being tested.
-- jsonb_path_query returns one row per match; report 2 has none.
SELECT id,
       jsonb_path_query(report, '$.suites[*].tests[*] ? (@.status == "failed").title')
         AS failed
FROM ci_reports
ORDER BY id;

-- Collect matches as an array, pass values in as variables, test with @?.
SELECT id,
       jsonb_path_query_array(report, '$.suites[*].tests[*].title') AS titles,
       jsonb_path_query_array(report, '$.suites[*].tests[*] ? (@.ms > $max).title',
                              '{"max": 500}')                       AS slow,
       report @? '$.suites[*].tests[*] ? (@.status == "failed")'   AS any_failed,
       jsonb_path_query_first(report, '$.suites.size()')             AS suites
FROM ci_reports
ORDER BY id;

-- lax mode (the default) unwraps arrays without [*] and skips missing keys.
SELECT id,
       jsonb_path_query_array(report, '$.suites.tests.title') AS unwrapped,
       jsonb_path_query_array(report, '$.suites.tests.error') AS errors
FROM ci_reports
ORDER BY id;

-- strict mode treats a missing key as an error.
SELECT jsonb_path_query_array(report, 'strict $.suites[*].tests[*].error')
FROM ci_reports
WHERE id = 1;
