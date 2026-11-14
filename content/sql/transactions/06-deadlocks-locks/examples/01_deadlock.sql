-- Two transfers at once: A moves 10 from a to b, B moves 20 from b to a.
-- Each locks its source row first.
A: BEGIN;
B: BEGIN;
A: UPDATE accounts SET balance = balance - 10 WHERE id = 'a';
B: UPDATE accounts SET balance = balance - 20 WHERE id = 'b';

-- Each now needs the row the other holds.
A: UPDATE accounts SET balance = balance + 10 WHERE id = 'b';
B: UPDATE accounts SET balance = balance + 20 WHERE id = 'a';

-- After deadlock_timeout (1 s) the detector finds the cycle and aborts one
-- transaction. The other one's update goes through.
A: ROLLBACK;
B: COMMIT;
A: SELECT id, balance FROM accounts ORDER BY id;
