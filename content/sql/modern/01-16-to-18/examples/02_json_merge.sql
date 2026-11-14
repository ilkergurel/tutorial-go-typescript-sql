CREATE TABLE test_cases (code text PRIMARY KEY, status text NOT NULL);
INSERT INTO test_cases VALUES ('TC-1', 'passed'), ('TC-2', 'failed'), ('TC-3', 'failed');

-- PG 16: IS JSON checks text before you cast it.
SELECT doc, doc IS JSON OBJECT AS is_object
FROM (VALUES ('{"case": "TC-1"}'), ('[1, 2]'), ('{"case": ')) AS v(doc)
ORDER BY doc;

-- A CI job reports its results as one JSON document.
CREATE TABLE ci_reports (id int PRIMARY KEY, body jsonb NOT NULL);
INSERT INTO ci_reports VALUES (1, '{"results": [
  {"case": "TC-1", "status": "passed"},
  {"case": "TC-2", "status": "passed"},
  {"case": "TC-4", "status": "failed"}]}');

-- PG 17: JSON_TABLE turns the document into rows.
SELECT r.*
FROM ci_reports,
     JSON_TABLE(body, '$.results[*]'
       COLUMNS (code text PATH '$.case', status text PATH '$.status')) AS r
ORDER BY r.code;

-- PG 17: MERGE with WHEN NOT MATCHED BY SOURCE and RETURNING merge_action();
-- PG 18: old/new in RETURNING.
WITH changes AS (
  MERGE INTO test_cases t
  USING (SELECT r.* FROM ci_reports,
           JSON_TABLE(body, '$.results[*]'
             COLUMNS (code text PATH '$.case', status text PATH '$.status')) AS r
         WHERE ci_reports.id = 1) AS r
  ON t.code = r.code
  WHEN MATCHED AND t.status <> r.status THEN UPDATE SET status = r.status
  WHEN NOT MATCHED THEN INSERT VALUES (r.code, r.status)
  WHEN NOT MATCHED BY SOURCE THEN UPDATE SET status = 'not run'
  RETURNING merge_action() AS action, t.code, old.status AS old_status, new.status AS new_status
)
SELECT * FROM changes ORDER BY code;
