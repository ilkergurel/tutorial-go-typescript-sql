-- Runtime sql-nontx: every statement commits on its own, as in psql, so the
-- table exists before we look at the locks a single ALTER takes.
CREATE TABLE accounts (id bigint PRIMARY KEY, email text NOT NULL);
INSERT INTO accounts SELECT g, 'user' || g || '@example.com' FROM generate_series(1, 1000) AS g;
CREATE VIEW my_locks AS
  SELECT mode FROM pg_locks
  WHERE pid = pg_backend_pid() AND locktype = 'relation' AND relation = 'accounts'::regclass;

BEGIN;
ALTER TABLE accounts ADD COLUMN country text;
SELECT mode FROM my_locks ORDER BY mode;
COMMIT;

BEGIN;
ALTER TABLE accounts ADD CONSTRAINT email_has_at CHECK (email LIKE '%@%') NOT VALID;
SELECT mode FROM my_locks ORDER BY mode;
COMMIT;

BEGIN;
ALTER TABLE accounts VALIDATE CONSTRAINT email_has_at;
SELECT mode FROM my_locks ORDER BY mode;
COMMIT;

BEGIN;
CREATE INDEX accounts_email ON accounts (email);
SELECT mode FROM my_locks ORDER BY mode;
ROLLBACK;

-- The online alternative builds the index without blocking writes, and for
-- exactly that reason it refuses to run inside a transaction block.
BEGIN;
CREATE INDEX CONCURRENTLY accounts_email ON accounts (email);
