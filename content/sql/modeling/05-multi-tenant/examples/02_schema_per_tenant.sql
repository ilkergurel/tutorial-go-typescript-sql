-- One schema per tenant with the same tables in each. search_path picks
-- the tenant, so queries carry no tenant_id at all.
CREATE SCHEMA tenant_kuzey;
CREATE SCHEMA tenant_guney;
CREATE TABLE tenant_kuzey.requirements (id int PRIMARY KEY, code text NOT NULL UNIQUE);
CREATE TABLE tenant_guney.requirements (id int PRIMARY KEY, code text NOT NULL UNIQUE);
INSERT INTO tenant_kuzey.requirements VALUES (1, 'REQ-1'), (2, 'REQ-2');
INSERT INTO tenant_guney.requirements VALUES (1, 'REQ-1');

-- Per request, inside its transaction. SET LOCAL ends with the
-- transaction, so a pooled connection can't carry it to the next request.
SET LOCAL search_path TO tenant_guney;
SELECT id, code FROM requirements ORDER BY id;

-- A report across tenants is one query per schema.
SELECT 'guney' AS tenant, count(*) AS requirements FROM tenant_guney.requirements
UNION ALL
SELECT 'kuzey', count(*) FROM tenant_kuzey.requirements
ORDER BY tenant;

-- Every migration runs once per schema. One that stopped halfway leaves
-- the tenants on different schemas.
ALTER TABLE tenant_kuzey.requirements ADD COLUMN priority smallint NOT NULL DEFAULT 3;
SELECT table_schema, count(*) AS columns
FROM information_schema.columns
WHERE table_schema LIKE 'tenant\_%' AND table_name = 'requirements'
GROUP BY table_schema
ORDER BY table_schema;

-- The new application code expects the column; Güney never got it.
SELECT id, code, priority FROM requirements ORDER BY id;
