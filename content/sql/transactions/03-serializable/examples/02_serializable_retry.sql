-- The same two transactions, now SERIALIZABLE. Nothing else changes.
A: BEGIN ISOLATION LEVEL SERIALIZABLE;
B: BEGIN ISOLATION LEVEL SERIALIZABLE;
A: SELECT count(*) AS on_call FROM doctors WHERE on_call;
B: SELECT count(*) AS on_call FROM doctors WHERE on_call;
A: UPDATE doctors SET on_call = false WHERE name = 'alice';
B: UPDATE doctors SET on_call = false WHERE name = 'bob';

-- No step waits. The first commit wins; the second could not have happened
-- in any serial order, so it fails with SQLSTATE 40001.
A: COMMIT;
B: COMMIT;

-- The retry: run the WHOLE transaction again, including the check.
B: BEGIN ISOLATION LEVEL SERIALIZABLE;
B: SELECT count(*) AS on_call FROM doctors WHERE on_call;
-- One left: the application refuses, and Bob stays on call.
B: ROLLBACK;

A: SELECT name, on_call FROM doctors ORDER BY name;
