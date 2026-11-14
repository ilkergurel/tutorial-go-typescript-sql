-- Runtime sql-nontx: one connection, several transactions, the way a pool
-- hands the same server session to request after request.
CREATE TABLE projects (id int PRIMARY KEY, tenant_id bigint NOT NULL, name text NOT NULL);
INSERT INTO projects VALUES (1, 1, 'Acme brakes'), (2, 2, 'Globex rail');
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE projects FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON projects
  USING (tenant_id = nullif(current_setting('app.tenant', true), '')::bigint);

-- Request 1 (tenant 1): the setting lives exactly as long as its transaction.
BEGIN;
SELECT set_config('app.tenant', '1', true);
SELECT name FROM projects;
COMMIT;

-- Request 2 forgot to set a tenant: the variable is still defined, but empty.
SELECT quote_literal(current_setting('app.tenant', true)) AS leftover,
       count(*) AS visible
FROM projects;

-- Request 3 (tenant 2) sets it for the whole session instead...
SELECT set_config('app.tenant', '2', false);
SELECT name FROM projects;
-- ...and request 4, someone else's, runs on the same connection without
-- setting a tenant. It inherits tenant 2.
SELECT name FROM projects;

-- The same policy without nullif: an empty leftover breaks the cast.
ALTER POLICY tenant_isolation ON projects
  USING (tenant_id = current_setting('app.tenant', true)::bigint);
RESET app.tenant;
SELECT name FROM projects;
