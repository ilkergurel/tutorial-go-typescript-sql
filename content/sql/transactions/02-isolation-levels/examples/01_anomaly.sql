-- LEARN(sql.transactions.isolation-levels): watch a non-repeatable read happen, then stop it.
-- sql-multisession: steps tagged A:/B: run on two connections in this order.

-- A: begin; read
A: BEGIN;
A: SELECT balance FROM accounts WHERE id = 'a';   -- 100

-- B: move 40 out, commit
B: BEGIN;
B: UPDATE accounts SET balance = balance - 40 WHERE id = 'a';
B: COMMIT;

-- A: read again in the SAME transaction
A: SELECT balance FROM accounts WHERE id = 'a';   -- 60 (non-repeatable read!)
A: COMMIT;

-- Prevent it: REPEATABLE READ pins one snapshot for the whole transaction.
A: BEGIN ISOLATION LEVEL REPEATABLE READ;
A: SELECT balance FROM accounts WHERE id = 'a';   -- 60

B: BEGIN;
B: UPDATE accounts SET balance = balance - 40 WHERE id = 'a';
B: COMMIT;

A: SELECT balance FROM accounts WHERE id = 'a';   -- still 60
A: COMMIT;
