-- Rule, checked in application code: at least one doctor stays on call.
-- Alice and Bob both feel ill and ask to go off call at the same moment.
A: BEGIN ISOLATION LEVEL REPEATABLE READ;
B: BEGIN ISOLATION LEVEL REPEATABLE READ;

-- Each checks the rule: two on call, so one may leave.
A: SELECT count(*) AS on_call FROM doctors WHERE on_call;
B: SELECT count(*) AS on_call FROM doctors WHERE on_call;

-- Each takes only themselves off: different rows, so no write conflict.
A: UPDATE doctors SET on_call = false WHERE name = 'alice';
B: UPDATE doctors SET on_call = false WHERE name = 'bob';
A: COMMIT;
B: COMMIT;

-- Both checks passed, and the rule is broken.
A: SELECT count(*) AS on_call FROM doctors WHERE on_call;
