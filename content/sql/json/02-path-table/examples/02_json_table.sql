CREATE TABLE ci_reports (
  id     int   PRIMARY KEY,
  report jsonb NOT NULL
);
INSERT INTO ci_reports VALUES
  (1, '{"build": 108, "suites": [
         {"name": "auth", "tests": [
            {"title": "Giriş", "status": "passed", "ms": 120},
            {"title": "Çıkış", "status": "failed", "ms": 85, "error": "timeout"}]},
         {"name": "export", "tests": []}]}'),
  (2, '{"build": 109, "suites": [
         {"name": "auth", "tests": [
            {"title": "Giriş", "status": "passed", "ms": "slow"},
            {"title": "Çıkış", "status": "passed"}]}]}');

-- One row per test: the row path picks the suites, NESTED PATH the tests
-- inside each suite, and every column has its own path and SQL type.
SELECT r.id, jt.*
FROM ci_reports AS r,
     JSON_TABLE(r.report, '$.suites[*]' COLUMNS (
       suite_no  FOR ORDINALITY,
       suite     text PATH '$.name',
       NESTED PATH '$.tests[*]' COLUMNS (
         test_no   FOR ORDINALITY,
         title     text PATH '$.title',
         ms        int  PATH '$.ms',
         has_error boolean EXISTS PATH '$.error'))) AS jt
ORDER BY r.id, jt.suite_no, jt.test_no;

-- For an import, be explicit: a missing ms gets a default, a wrong type fails.
CREATE TABLE test_results (build int, suite text, title text, ms int);
INSERT INTO test_results
SELECT jt.*
FROM ci_reports AS r,
     JSON_TABLE(r.report, '$' COLUMNS (
       build int PATH '$.build',
       NESTED PATH '$.suites[*]' COLUMNS (
         suite text PATH '$.name',
         NESTED PATH '$.tests[*]' COLUMNS (
           title text PATH '$.title',
           ms    int  PATH '$.ms' DEFAULT 0 ON EMPTY ERROR ON ERROR)))) AS jt;
