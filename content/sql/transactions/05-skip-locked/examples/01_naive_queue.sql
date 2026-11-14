-- Two workers, each in its own transaction: lock the oldest queued job,
-- do the work, mark it done, commit.
A: BEGIN;
A: SELECT id, kind FROM jobs WHERE status = 'queued' ORDER BY id LIMIT 1 FOR UPDATE;

-- B asks for the oldest queued job too, finds it locked, and queues behind A.
B: BEGIN;
B: SELECT id, kind FROM jobs WHERE status = 'queued' ORDER BY id LIMIT 1 FOR UPDATE;

-- B sits idle for as long as A's job runs.
A: UPDATE jobs SET status = 'done' WHERE id = 1;
A: COMMIT;

B: UPDATE jobs SET status = 'done' WHERE id = 2;
B: COMMIT;
