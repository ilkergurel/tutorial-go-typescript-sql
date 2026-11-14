-- A does what sqlrun.Run does to every lesson: DDL and data in one open transaction.
A: BEGIN;
A: CREATE TABLE scratch (id int);
A: INSERT INTO scratch VALUES (1), (2);

-- B shares the schema (same search_path), but catalog rows are MVCC rows too.
B: SELECT to_regclass('scratch') AS scratch;

-- B claims the same name: the catalog's unique index waits for A's outcome.
B: CREATE TABLE scratch (id int);
A: ROLLBACK;

-- A's table and rows never existed for anyone; the table now is B's, empty.
A: SELECT count(*) AS rows_from_a FROM scratch;
