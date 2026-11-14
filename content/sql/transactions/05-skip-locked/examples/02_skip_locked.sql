-- The same workers, with SKIP LOCKED: a locked job is someone else's.
A: BEGIN;
A: SELECT id, kind FROM jobs WHERE status = 'queued'
   ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED;

B: BEGIN;
B: SELECT id, kind FROM jobs WHERE status = 'queued'
   ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED;
B: UPDATE jobs SET status = 'done' WHERE id = 2;
B: COMMIT;

-- B's next job: 1 is still locked by A, 2 is done.
B: BEGIN;
B: SELECT id, kind FROM jobs WHERE status = 'queued'
   ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED;

-- A's process dies mid-job. The server rolls its transaction back,
-- which releases the lock: job 1 was never marked done.
A: ROLLBACK;
B: UPDATE jobs SET status = 'done' WHERE id = 3;
B: COMMIT;

B: BEGIN;
B: SELECT id, kind FROM jobs WHERE status = 'queued'
   ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED;
B: UPDATE jobs SET status = 'done' WHERE id = 1;
B: COMMIT;
