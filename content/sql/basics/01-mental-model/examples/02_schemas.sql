-- Schemas are namespaces inside one database: the same table name can
-- exist once per schema.
CREATE SCHEMA billing;
CREATE SCHEMA support;
CREATE TABLE billing.tickets (id int, note text);
CREATE TABLE support.tickets (id int, note text);
INSERT INTO billing.tickets VALUES (1, 'invoice 2026-11 is overdue');
INSERT INTO support.tickets VALUES (1, 'login fails for İlknur');

-- An unqualified name resolves through the search path: first match wins.
SET LOCAL search_path TO support, billing;
SELECT note FROM tickets;

SET LOCAL search_path TO billing, support;
SELECT note FROM tickets;

-- CREATE without a schema lands in the first schema on the path.
CREATE TABLE refunds (id int);
SELECT table_schema, table_name
FROM information_schema.tables
WHERE table_schema IN ('billing', 'support')
ORDER BY table_schema, table_name;

-- Qualified names ignore the path entirely.
SELECT 'support' AS schema, note FROM support.tickets
UNION ALL
SELECT 'billing', note FROM billing.tickets
ORDER BY schema;
