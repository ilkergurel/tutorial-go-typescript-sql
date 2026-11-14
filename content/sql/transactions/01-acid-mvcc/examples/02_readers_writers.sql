-- sql-multisession: A: and B: are two connections, steps run in this order.
-- A moves 40 from a to b and hasn't committed yet.
A: BEGIN;
A: UPDATE accounts SET balance = balance - 40 WHERE id = 'a';
A: UPDATE accounts SET balance = balance + 40 WHERE id = 'b';

-- B reads both rows mid-transfer: no waiting, and no half-done transfer.
B: SELECT id, balance FROM accounts ORDER BY id;

-- B writes a row that A has written: this has to wait for A.
B: UPDATE accounts SET balance = balance + 1 WHERE id = 'a';

A: COMMIT;

-- B's update was applied to A's committed version: 60 + 1.
B: SELECT id, balance FROM accounts ORDER BY id;
