-- One failed statement aborts the whole transaction.
A: BEGIN;
A: INSERT INTO accounts VALUES ('c', 10);
A: INSERT INTO accounts VALUES ('a', 0);
A: SELECT count(*) FROM accounts;
A: COMMIT;

-- A savepoint limits the damage to the statements after it.
A: BEGIN;
A: INSERT INTO accounts VALUES ('c', 10);
A: SAVEPOINT try_a;
A: INSERT INTO accounts VALUES ('a', 0);
A: ROLLBACK TO SAVEPOINT try_a;
A: INSERT INTO accounts VALUES ('d', 20);
A: COMMIT;

-- B, another connection, sees exactly what was committed.
B: SELECT id, balance FROM accounts ORDER BY id;
