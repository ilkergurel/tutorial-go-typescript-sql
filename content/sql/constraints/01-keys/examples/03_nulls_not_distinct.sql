-- tenant_id NULL means "global default"; a tenant's own row overrides it.
CREATE TABLE settings_loose (
  tenant_id int,
  key       text NOT NULL,
  value     text NOT NULL,
  UNIQUE (tenant_id, key)
);
INSERT INTO settings_loose VALUES
  (NULL, 'theme', 'light'),
  (NULL, 'theme', 'dark'),   -- accepted: NULL is not equal to NULL
  (7,    'theme', 'dark');
SELECT tenant_id, key, value
FROM settings_loose
ORDER BY tenant_id NULLS FIRST, value;

-- PG 15+: treat NULLs as equal for this constraint only.
CREATE TABLE settings (
  tenant_id int,
  key       text NOT NULL,
  value     text NOT NULL,
  CONSTRAINT settings_scope_key UNIQUE NULLS NOT DISTINCT (tenant_id, key)
);
INSERT INTO settings VALUES (NULL, 'theme', 'light'), (7, 'theme', 'dark');
INSERT INTO settings VALUES (NULL, 'theme', 'dark');
