-- LEARN(sql.security-ops.rls): the tenant boundary is enforced by the table, not by each query.
CREATE TABLE projects (
  id        int PRIMARY KEY,
  tenant_id bigint NOT NULL,
  name      text NOT NULL
);
INSERT INTO projects VALUES
  (1, 1, 'Acme brake controller'),
  (2, 1, 'Acme pump firmware'),
  (3, 2, 'Globex rail signalling');

-- The app sets app.tenant per transaction. Unset or empty gives NULL,
-- and NULL = anything is never true: no tenant, no rows.
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON projects
  USING (tenant_id = nullif(current_setting('app.tenant', true), '')::bigint);

-- This session owns the table, and owners skip policies...
SELECT count(*) AS seen_by_owner FROM projects;
-- ...unless the table forces them on its owner as well.
ALTER TABLE projects FORCE ROW LEVEL SECURITY;
SELECT count(*) AS no_tenant_set FROM projects;

SELECT set_config('app.tenant', '1', true);
SELECT id, name FROM projects ORDER BY id;
-- The policy is a predicate the planner adds to every query on the table.
EXPLAIN (COSTS OFF) SELECT id, name FROM projects;

-- Rows you cannot see cannot be changed: no error, just nothing matched.
UPDATE projects SET name = 'renamed by Acme' WHERE id = 3;
DELETE FROM projects WHERE id = 3;

SELECT set_config('app.tenant', '2', true);
SELECT id, name FROM projects ORDER BY id;

-- A new row must pass the policy too: USING doubles as WITH CHECK.
INSERT INTO projects VALUES (4, 1, 'planted by Globex');
