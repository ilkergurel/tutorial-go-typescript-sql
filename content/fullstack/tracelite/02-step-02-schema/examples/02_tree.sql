-- The tables of 001_init.sql that a tree needs, abridged to the tree's columns.
CREATE TABLE projects (id uuid PRIMARY KEY DEFAULT uuidv7(), name text NOT NULL);
CREATE TABLE requirements (
  id         uuid PRIMARY KEY DEFAULT uuidv7(),
  project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
  parent_id  uuid,
  code       text COLLATE "C" NOT NULL,
  CONSTRAINT requirements_code_key UNIQUE (project_id, code),
  CONSTRAINT requirements_in_project UNIQUE (project_id, id),
  CONSTRAINT requirements_parent_fkey FOREIGN KEY (project_id, parent_id)
    REFERENCES requirements (project_id, id)
);
CREATE INDEX requirements_parent ON requirements (parent_id);
INSERT INTO projects (name) VALUES ('Ödeme'), ('Raporlama');
INSERT INTO requirements (project_id, code)
SELECT id, unnest(ARRAY['AUTH', 'EXPORT']) FROM projects WHERE name = 'Ödeme'
UNION ALL SELECT id, 'REPORT' FROM projects WHERE name = 'Raporlama';
INSERT INTO requirements (project_id, parent_id, code)  -- one INSERT per level
SELECT p.project_id, p.id, v.code FROM requirements p
JOIN (VALUES ('AUTH', 'AUTH-LOGIN'), ('AUTH', 'AUTH-RESET'), ('EXPORT', 'EXPORT-CSV'),
             ('REPORT', 'REPORT-PDF')) AS v (parent, code) ON p.code = v.parent;
INSERT INTO requirements (project_id, parent_id, code)
SELECT p.project_id, p.id, v.code FROM requirements p
JOIN (VALUES ('AUTH-LOGIN', 'AUTH-LOGIN-2FA'), ('AUTH-LOGIN', 'AUTH-LOGIN-LOCKOUT'))
  AS v (parent, code) ON p.code = v.parent;

-- The queries take a project name and codes so they read well here; the
-- store's versions take ids.
-- GET /api/projects/{projectId}/requirement-tree: the whole tree, depth first.
PREPARE tree(text) AS
WITH RECURSIVE tree AS (
  SELECT r.id, r.code, 0 AS depth FROM requirements r JOIN projects p ON p.id = r.project_id
  WHERE p.name = $1 AND r.parent_id IS NULL
  UNION ALL
  SELECT c.id, c.code, t.depth + 1 FROM requirements c JOIN tree t ON c.parent_id = t.id
) SEARCH DEPTH FIRST BY code SET ord
SELECT repeat('. ', depth) || code AS outline, depth FROM tree ORDER BY ord;
EXECUTE tree('Ödeme');

-- GET /api/projects/{projectId}/requirements: a flat page by code, after the
-- cursor. The store's exact statement, planned without values:
EXPLAIN (GENERIC_PLAN, COSTS OFF)
SELECT id, code FROM requirements WHERE project_id = $1 AND code > $2 ORDER BY code LIMIT $3;
SELECT r.code FROM requirements r JOIN projects p ON p.id = r.project_id  -- page size 2
WHERE p.name = 'Ödeme' AND r.code > 'AUTH-LOGIN' ORDER BY r.code LIMIT 2 + 1;

-- Moving $1 under $2 closes a loop if $1 is $2 or one of $2's ancestors.
PREPARE would_cycle(text, text) AS
WITH RECURSIVE up AS (
  SELECT id, parent_id FROM requirements WHERE code = $2
  UNION ALL
  SELECT r.id, r.parent_id FROM requirements r JOIN up ON r.id = up.parent_id
) CYCLE id SET is_cycle USING path
SELECT $1 AS moved, $2 AS new_parent,
       EXISTS (SELECT FROM up JOIN requirements m USING (id) WHERE m.code = $1) AS would_cycle;
EXECUTE would_cycle('AUTH-RESET', 'AUTH-LOGIN');
EXECUTE would_cycle('AUTH', 'AUTH-LOGIN-2FA');

-- Nothing in the schema refuses that second move: every row still has a valid parent.
UPDATE requirements SET parent_id = (SELECT id FROM requirements WHERE code = 'AUTH-LOGIN-2FA')
WHERE code = 'AUTH';
EXECUTE tree('Ödeme');

-- A project goes with its whole tree; one requirement doesn't go while it has children.
DELETE FROM projects WHERE name = 'Raporlama';
SELECT count(*) AS requirements_left FROM requirements;
DELETE FROM requirements WHERE code = 'AUTH-LOGIN';
