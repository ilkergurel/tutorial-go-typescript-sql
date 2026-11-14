-- psql patterns follow SQL's identifier rules, so look at the names the
-- catalog actually stores before blaming \dt for "missing" tables.
CREATE TABLE Payments (id int);     -- unquoted: folded to lower case
CREATE TABLE "Invoices" (id int);   -- quoted: stored exactly as written
CREATE TABLE Şubeler (id int);      -- unquoted, but Ş is not ASCII

-- 55 characters, 64 bytes: one byte over the identifier limit.
CREATE TABLE iade_talepleri_için_günlük_özet_çizelgesi_ağustos_ölçüm (id int);

SELECT c.relname AS stored_name, length(c.relname) AS chars,
       octet_length(c.relname) AS bytes
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = current_schema() AND c.relkind = 'r'
ORDER BY c.relname;

-- to_regclass returns NULL instead of failing: a quick "does it exist?".
SELECT to_regclass('payments')   IS NOT NULL AS payments,
       to_regclass('PAYMENTS')   IS NOT NULL AS "PAYMENTS",
       to_regclass('invoices')   IS NOT NULL AS invoices,
       to_regclass('"Invoices"') IS NOT NULL AS "quoted Invoices",
       to_regclass('şubeler')    IS NOT NULL AS şubeler;

-- Forget the quotes once and the table "does not exist".
SELECT count(*) FROM Invoices;
