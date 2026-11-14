-- Column tenancy enforced by the database: row-level security.
CREATE TABLE requirements (
  tenant_id int  NOT NULL DEFAULT current_setting('app.tenant')::int,
  id        int  PRIMARY KEY,
  code      text NOT NULL,
  UNIQUE (tenant_id, code)
);
INSERT INTO requirements VALUES (1, 1, 'REQ-1'), (1, 2, 'REQ-2'), (2, 3, 'REQ-1');

ALTER TABLE requirements ENABLE ROW LEVEL SECURITY;
-- The table owner bypasses policies unless they are forced on it too.
ALTER TABLE requirements FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON requirements
  USING (tenant_id = current_setting('app.tenant')::int);

-- Superusers and BYPASSRLS roles skip every policy. The app role must not.
SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user;

-- Per request, inside its transaction: true means "local to it".
SELECT set_config('app.tenant', '2', true);

-- No tenant filter anywhere below: the policy adds it.
SELECT tenant_id, id, code FROM requirements ORDER BY id;
UPDATE requirements SET code = 'REQ-9' WHERE id = 1;     -- tenant 1's row
INSERT INTO requirements (id, code) VALUES (4, 'REQ-2'); -- tenant_id from the setting
SELECT tenant_id, id, code FROM requirements ORDER BY id;

-- A row for another tenant fails the policy's check.
INSERT INTO requirements (tenant_id, id, code) VALUES (1, 5, 'REQ-3');
