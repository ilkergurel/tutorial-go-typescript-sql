-- queries/report.sql: one row per requirement and linked test, in tree order, so
-- the rows of one requirement are adjacent. The tables and project Ödeme are in
-- this example's setup.
PREPARE report (uuid) AS
WITH RECURSIVE tree AS (
  SELECT id, code, title, 0 AS depth FROM requirements WHERE project_id = $1 AND parent_id IS NULL
  UNION ALL
  SELECT c.id, c.code, c.title, t.depth + 1 FROM requirements c JOIN tree t ON c.parent_id = t.id
) SEARCH DEPTH FIRST BY code SET ord
SELECT t.code, t.title, t.depth,
  EXISTS (SELECT FROM requirements k WHERE k.parent_id = t.id) AS has_children,
  CASE WHEN count(tc.id) OVER w = 0 THEN 'uncovered'
       WHEN count(*) FILTER (WHERE last.result = 'failed') OVER w > 0 THEN 'failing'
       WHEN count(*) FILTER (WHERE last.result = 'passed') OVER w = count(*) OVER w THEN 'passing'
       ELSE 'incomplete' END AS coverage,
  tc.code AS test, last.result, last.finished_at
FROM tree t
LEFT JOIN trace_links l ON l.project_id = $1 AND l.requirement_id = t.id
LEFT JOIN test_cases tc ON tc.id = l.test_case_id
LEFT JOIN LATERAL (SELECT result, finished_at FROM test_runs WHERE test_case_id = tc.id
                   ORDER BY finished_at DESC, id DESC LIMIT 1) last ON true
WINDOW w AS (PARTITION BY t.id)
ORDER BY t.ord, tc.code;

EXECUTE report('01937b6e-0000-7000-8000-00000000000a');
