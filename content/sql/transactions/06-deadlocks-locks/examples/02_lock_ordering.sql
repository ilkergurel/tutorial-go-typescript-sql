-- The same two transfers. Each first locks BOTH rows, in key order,
-- whichever direction the money flows.
A: BEGIN;
B: BEGIN;
A: SELECT id FROM accounts WHERE id IN ('a', 'b') ORDER BY id FOR NO KEY UPDATE;
B: SELECT id FROM accounts WHERE id IN ('a', 'b') ORDER BY id FOR NO KEY UPDATE;

-- B waits for row a while holding nothing, so no cycle can form.
A: UPDATE accounts SET balance = balance - 10 WHERE id = 'a';
A: UPDATE accounts SET balance = balance + 10 WHERE id = 'b';
A: COMMIT;

B: UPDATE accounts SET balance = balance - 20 WHERE id = 'b';
B: UPDATE accounts SET balance = balance + 20 WHERE id = 'a';
B: COMMIT;
B: SELECT id, balance FROM accounts ORDER BY id;
