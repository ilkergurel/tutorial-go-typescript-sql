-- payments.account_id REFERENCES accounts. Inserting a payment makes
-- Postgres take FOR KEY SHARE on the account row: "don't delete it, don't
-- change its key until I commit".

-- A plain UPDATE of a non-key column takes FOR NO KEY UPDATE.
A: BEGIN;
A: UPDATE accounts SET balance = balance - 10 WHERE id = 'a';
B: INSERT INTO payments (account_id, amount) VALUES ('a', 10);
A: COMMIT;

-- SELECT ... FOR UPDATE is stronger than the update needs.
A: BEGIN;
A: SELECT balance FROM accounts WHERE id = 'a' FOR UPDATE;
B: INSERT INTO payments (account_id, amount) VALUES ('a', 20);
A: UPDATE accounts SET balance = balance - 20 WHERE id = 'a';
A: COMMIT;

-- FOR NO KEY UPDATE locks exactly what the UPDATE will lock anyway.
A: BEGIN;
A: SELECT balance FROM accounts WHERE id = 'a' FOR NO KEY UPDATE;
B: INSERT INTO payments (account_id, amount) VALUES ('a', 30);
A: UPDATE accounts SET balance = balance - 30 WHERE id = 'a';
A: COMMIT;
