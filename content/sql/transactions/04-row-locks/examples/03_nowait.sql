-- A holds the row lock. B tries three ways to get at the row.
A: BEGIN;
A: SELECT balance FROM accounts WHERE id = 'a' FOR UPDATE;

-- NOWAIT: fail at once instead of queueing behind A.
B: SELECT balance FROM accounts WHERE id = 'a' FOR UPDATE NOWAIT;

-- lock_timeout: wait, but only this long. It covers every lock the
-- statement needs, including the row lock a plain UPDATE takes.
B: SET lock_timeout = '100ms';
B: UPDATE accounts SET balance = 0 WHERE id = 'a';

-- A plain read never waits for a row lock.
B: SELECT balance FROM accounts WHERE id = 'a';
A: COMMIT;
