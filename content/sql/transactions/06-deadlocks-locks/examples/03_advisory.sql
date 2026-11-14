-- Advisory locks lock a number, not a row. 4242 means "the nightly report"
-- here; the database doesn't know or care.

-- Transaction-level: released by COMMIT or ROLLBACK, like a row lock.
A: BEGIN;
A: SELECT pg_try_advisory_xact_lock(4242) AS got_it;
B: SELECT pg_try_advisory_xact_lock(4242) AS got_it;
A: COMMIT;
B: SELECT pg_try_advisory_xact_lock(4242) AS got_it;

-- Session-level: held until unlocked or the connection closes,
-- and transactions don't touch it.
A: BEGIN;
A: SELECT pg_try_advisory_lock(4242) AS got_it;
A: ROLLBACK;
B: SELECT pg_try_advisory_lock(4242) AS got_it;
A: SELECT pg_advisory_unlock(4242) AS released;
B: SELECT pg_try_advisory_lock(4242) AS got_it;
B: SELECT pg_advisory_unlock(4242) AS released;
