-- LEARN(sql.modeling.multi-tenant): the tenant is part of every key, so no row can point across tenants.
CREATE TABLE tenants (id int PRIMARY KEY, name text NOT NULL);
CREATE TABLE projects (
  id        int  PRIMARY KEY,
  tenant_id int  NOT NULL REFERENCES tenants,
  name      text NOT NULL,
  UNIQUE (tenant_id, id)                   -- the target for tenant-aware FKs
);
INSERT INTO tenants VALUES (1, 'Kuzey'), (2, 'Güney');
INSERT INTO projects VALUES (10, 1, 'Checkout'), (20, 2, 'Checkout');

-- Naive: the FK proves project 10 exists, not that it is Güney's.
CREATE TABLE requirements_naive (
  id         int  PRIMARY KEY,
  tenant_id  int  NOT NULL REFERENCES tenants,
  project_id int  NOT NULL REFERENCES projects,
  code       text NOT NULL
);
INSERT INTO requirements_naive VALUES (1, 2, 10, 'REQ-1');   -- accepted

-- Tenant-aware: the FK includes tenant_id, and so does uniqueness.
CREATE TABLE requirements (
  id         int  PRIMARY KEY,
  tenant_id  int  NOT NULL REFERENCES tenants,
  project_id int  NOT NULL,
  code       text NOT NULL,
  UNIQUE (tenant_id, code),
  FOREIGN KEY (tenant_id, project_id) REFERENCES projects (tenant_id, id)
);
CREATE INDEX ON requirements (tenant_id, project_id);
INSERT INTO requirements VALUES (1, 1, 10, 'REQ-1'), (2, 2, 20, 'REQ-1');

-- Every query filters by tenant. Forget it once, and the page shows both.
SELECT id, code, project_id FROM requirements WHERE tenant_id = 2 ORDER BY id;
SELECT tenant_id, id, code FROM requirements ORDER BY id;

-- Güney's requirement in Kuzey's project: (2, 10) is not a project.
INSERT INTO requirements VALUES (3, 2, 10, 'REQ-2');
