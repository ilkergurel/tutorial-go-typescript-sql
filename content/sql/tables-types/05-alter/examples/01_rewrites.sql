CREATE TABLE accounts (
  id   integer PRIMARY KEY,
  name varchar(100) NOT NULL
);
CREATE INDEX accounts_name ON accounts (name);
INSERT INTO accounts SELECT g, 'Kullanıcı ' || g FROM generate_series(1, 1000) AS g;

-- Table and index data live in files named by relfilenode. A rewrite copies
-- everything into new files, so the numbers change. Log them after each step.
CREATE FUNCTION files(OUT tbl oid, OUT idx oid) LANGUAGE sql
  AS $$ SELECT pg_relation_filenode('accounts'), pg_relation_filenode('accounts_name') $$;
CREATE TABLE steps (n int GENERATED ALWAYS AS IDENTITY, change text, tbl oid, idx oid);
INSERT INTO steps (change, tbl, idx) SELECT '(start)', tbl, idx FROM files();

ALTER TABLE accounts ADD COLUMN note text;
INSERT INTO steps (change, tbl, idx) SELECT 'add column, no default', tbl, idx FROM files();
ALTER TABLE accounts ADD COLUMN plan text NOT NULL DEFAULT 'free';
INSERT INTO steps (change, tbl, idx) SELECT 'add column, constant default', tbl, idx FROM files();
ALTER TABLE accounts ADD COLUMN token uuid NOT NULL DEFAULT uuidv7();
INSERT INTO steps (change, tbl, idx) SELECT 'add column, volatile default', tbl, idx FROM files();
ALTER TABLE accounts ADD COLUMN plan_label text GENERATED ALWAYS AS (upper(plan));
INSERT INTO steps (change, tbl, idx) SELECT 'add virtual generated column', tbl, idx FROM files();
ALTER TABLE accounts ALTER COLUMN name TYPE varchar(200);
INSERT INTO steps (change, tbl, idx) SELECT 'varchar(100) -> varchar(200)', tbl, idx FROM files();
ALTER TABLE accounts ALTER COLUMN name TYPE text;
INSERT INTO steps (change, tbl, idx) SELECT 'varchar(200) -> text', tbl, idx FROM files();
ALTER TABLE accounts ALTER COLUMN name TYPE text COLLATE "tr-x-icu";
INSERT INTO steps (change, tbl, idx) SELECT 'collation -> tr-x-icu', tbl, idx FROM files();
ALTER TABLE accounts ALTER COLUMN id TYPE bigint;
INSERT INTO steps (change, tbl, idx) SELECT 'integer -> bigint', tbl, idx FROM files();
ALTER TABLE accounts DROP COLUMN note;
INSERT INTO steps (change, tbl, idx) SELECT 'drop column', tbl, idx FROM files();

SELECT change,
       tbl <> lag(tbl) OVER (ORDER BY n) AS table_rewritten,
       idx <> lag(idx) OVER (ORDER BY n) AS name_index_rebuilt
FROM steps
ORDER BY n;
