-- Claim in one short transaction, work outside it, finish in another.
-- A's claim is still open when B claims, so B skips job 1.
A: BEGIN;
A: UPDATE jobs SET status = 'running', attempts = attempts + 1,
                  lease_until = now() + interval '5 minutes'
   WHERE id = (SELECT id FROM jobs WHERE status = 'queued'
               ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED)
   RETURNING id, kind, attempts;
B: UPDATE jobs SET status = 'running', attempts = attempts + 1,
                  lease_until = now() + interval '5 minutes'
   WHERE id = (SELECT id FROM jobs WHERE status = 'queued'
               ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED)
   RETURNING id, kind, attempts;
A: COMMIT;

-- B does the work, then finishes in a second short transaction.
B: UPDATE jobs SET status = 'done', lease_until = NULL WHERE id = 2;

-- A dies while working. Its claim is committed, so nothing rolls back:
-- job 1 stays 'running' until its lease runs out. (Demo only: expire it now.)
B: UPDATE jobs SET lease_until = now() - interval '1 second' WHERE id = 1;

-- The reaper, run by any worker now and then, puts expired jobs back.
B: UPDATE jobs SET status = 'queued', lease_until = NULL
   WHERE status = 'running' AND lease_until < now()
   RETURNING id, attempts;

-- The next claim gets job 1 again; attempts says it is the second try.
B: UPDATE jobs SET status = 'running', attempts = attempts + 1,
                  lease_until = now() + interval '5 minutes'
   WHERE id = (SELECT id FROM jobs WHERE status = 'queued'
               ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED)
   RETURNING id, kind, attempts;
