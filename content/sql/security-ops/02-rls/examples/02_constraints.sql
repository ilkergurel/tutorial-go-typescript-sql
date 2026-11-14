CREATE TABLE projects (
  id        int PRIMARY KEY,
  tenant_id bigint NOT NULL,
  code      text NOT NULL UNIQUE,
  name      text NOT NULL
);
CREATE TABLE requirements (
  id         int PRIMARY KEY,
  tenant_id  bigint NOT NULL,
  project_id int NOT NULL REFERENCES projects,
  title      text NOT NULL
);
INSERT INTO projects VALUES (1, 1, 'BRK', 'Acme brakes'), (3, 2, 'RAIL', 'Globex rail');

ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE projects FORCE ROW LEVEL SECURITY;
ALTER TABLE requirements ENABLE ROW LEVEL SECURITY;
ALTER TABLE requirements FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON projects
  USING (tenant_id = nullif(current_setting('app.tenant', true), '')::bigint);
CREATE POLICY tenant_isolation ON requirements
  USING (tenant_id = nullif(current_setting('app.tenant', true), '')::bigint);

SELECT set_config('app.tenant', '1', true);
SELECT code FROM projects;

-- The unique index sees every tenant's rows: "nothing inserted" means
-- some other tenant already uses the code RAIL.
INSERT INTO projects VALUES (10, 1, 'RAIL', 'Acme rail test rig')
  ON CONFLICT (code) DO NOTHING;
-- The foreign key check sees them too: Acme can attach to a Globex project.
INSERT INTO requirements VALUES (100, 1, 3, 'Stops within 300 m');
SELECT r.title, p.name AS project FROM requirements r LEFT JOIN projects p ON p.id = r.project_id;

-- Fix: make the tenant part of every key, so constraints stay inside one tenant.
DELETE FROM requirements;
ALTER TABLE requirements DROP CONSTRAINT requirements_project_id_fkey;
ALTER TABLE projects DROP CONSTRAINT projects_code_key,
  ADD UNIQUE (tenant_id, code),
  ADD UNIQUE (tenant_id, id);
ALTER TABLE requirements
  ADD FOREIGN KEY (tenant_id, project_id) REFERENCES projects (tenant_id, id);

INSERT INTO projects VALUES (10, 1, 'RAIL', 'Acme rail test rig');
INSERT INTO requirements VALUES (101, 1, 3, 'Stops within 300 m');
