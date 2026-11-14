-- Read, compute in the application, write the result back: two withdrawals
-- (A takes 30, B takes 50) from account a, which holds 100.
A: BEGIN;
B: BEGIN;
A: SELECT balance FROM accounts WHERE id = 'a';
B: SELECT balance FROM accounts WHERE id = 'a';
A: UPDATE accounts SET balance = 70 WHERE id = 'a';
A: COMMIT;
B: UPDATE accounts SET balance = 50 WHERE id = 'a';
B: COMMIT;

-- The same two withdrawals from account b, reading with FOR UPDATE.
A: BEGIN;
B: BEGIN;
A: SELECT balance FROM accounts WHERE id = 'b' FOR UPDATE;
B: SELECT balance FROM accounts WHERE id = 'b' FOR UPDATE;
A: UPDATE accounts SET balance = 70 WHERE id = 'b';
A: COMMIT;
-- B's read returned 70 once A committed, so B computes 70 - 50.
B: UPDATE accounts SET balance = 20 WHERE id = 'b';
B: COMMIT;

-- Account a lost A's withdrawal; account b has both.
A: SELECT id, balance FROM accounts ORDER BY id;
