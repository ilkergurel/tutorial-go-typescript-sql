-- Requirements and test cases, abridged to what links need.
CREATE TABLE projects (id uuid PRIMARY KEY DEFAULT uuidv7(), name text NOT NULL);
CREATE TABLE requirements (id uuid PRIMARY KEY DEFAULT uuidv7(), code text COLLATE "C" NOT NULL,
  project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE, UNIQUE (project_id, id));
CREATE TABLE test_cases (id uuid PRIMARY KEY DEFAULT uuidv7(), code text COLLATE "C" NOT NULL,
  project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE, UNIQUE (project_id, id));
-- The rest of 001_init.sql, verbatim.
CREATE TABLE trace_links (  -- "this test case verifies this requirement"
  project_id     uuid NOT NULL,
  requirement_id uuid NOT NULL,
  test_case_id   uuid NOT NULL,
  created_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, requirement_id, test_case_id),
  CONSTRAINT trace_links_requirement_fkey FOREIGN KEY (project_id, requirement_id)
    REFERENCES requirements (project_id, id) ON DELETE CASCADE,
  CONSTRAINT trace_links_test_case_fkey FOREIGN KEY (project_id, test_case_id)
    REFERENCES test_cases (project_id, id) ON DELETE CASCADE
);
CREATE INDEX trace_links_test_case ON trace_links (project_id, test_case_id);
CREATE TABLE test_runs (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  test_case_id uuid NOT NULL CONSTRAINT test_runs_test_case_fkey REFERENCES test_cases ON DELETE RESTRICT,
  result       text NOT NULL CONSTRAINT test_runs_result_valid CHECK (result IN ('passed', 'failed', 'skipped')),
  finished_at  timestamptz NOT NULL DEFAULT now(),
  duration_ms  integer CONSTRAINT test_runs_duration_ms_valid CHECK (duration_ms >= 0),
  notes        text NOT NULL DEFAULT ''
);
CREATE INDEX test_runs_latest ON test_runs (test_case_id, finished_at DESC, id DESC);
-- end of 001_init.sql

INSERT INTO projects (name) VALUES ('Ödeme');
INSERT INTO requirements (project_id, code)
SELECT id, unnest(ARRAY['AUTH-LOGIN', 'AUTH-LOCKOUT', 'AUTH-RESET', 'EXPORT-CSV']) FROM projects;
INSERT INTO test_cases (project_id, code) SELECT id, unnest(ARRAY['TC-1', 'TC-2', 'TC-3', 'TC-4']) FROM projects;

-- PUT /api/requirements/{requirementId}/test-cases/{testCaseId}: link, idempotently.
PREPARE link(text, text) AS
INSERT INTO trace_links (project_id, requirement_id, test_case_id)
SELECT r.project_id, r.id, t.id FROM requirements r, test_cases t WHERE r.code = $1 AND t.code = $2
ON CONFLICT DO NOTHING;
EXECUTE link('AUTH-LOGIN', 'TC-1'); EXECUTE link('AUTH-LOGIN', 'TC-2');
EXECUTE link('AUTH-LOCKOUT', 'TC-1'); EXECUTE link('AUTH-LOCKOUT', 'TC-4');
EXECUTE link('AUTH-RESET', 'TC-3');
EXECUTE link('AUTH-LOGIN', 'TC-1');  -- again: nothing happens

INSERT INTO test_runs (test_case_id, result, finished_at, duration_ms)
SELECT t.id, v.result, v.at::timestamptz, v.ms FROM test_cases t
JOIN (VALUES ('TC-1', 'failed', '2026-11-12 10:00', 900), ('TC-1', 'passed', '2026-11-13 09:00', 850),
             ('TC-2', 'passed', '2026-11-13 09:01', 120), ('TC-3', 'passed', '2026-11-12 10:00', 300),
             ('TC-3', 'failed', '2026-11-13 09:02', 310)) AS v (code, result, at, ms) ON t.code = v.code;

-- Coverage per requirement: what the latest run of each linked test says.
SELECT r.code, count(l.test_case_id) AS tests,
       bool_and(last.result = 'passed') AS naive_all_passed,
       CASE WHEN count(l.test_case_id) = 0 THEN 'uncovered'
            WHEN bool_or(last.result = 'failed') THEN 'failing'
            WHEN count(*) FILTER (WHERE last.result = 'passed') = count(*) THEN 'passing'
            ELSE 'incomplete' END AS coverage
FROM requirements r
LEFT JOIN trace_links l ON l.project_id = r.project_id AND l.requirement_id = r.id
LEFT JOIN LATERAL (SELECT result FROM test_runs WHERE test_case_id = l.test_case_id
                   ORDER BY finished_at DESC, id DESC LIMIT 1) AS last ON true
GROUP BY r.id ORDER BY r.code;

-- The project's delete cascades to requirements, test cases and links, and stops at runs.
DELETE FROM projects WHERE name = 'Ödeme';
