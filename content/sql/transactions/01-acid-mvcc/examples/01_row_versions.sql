-- Runtime sql: this whole script is one transaction, rolled back at the end.
CREATE TABLE accounts (id text PRIMARY KEY, balance int NOT NULL);
INSERT INTO accounts VALUES ('a', 100), ('b', 0);

-- Hidden system columns on every row version: ctid = (page, slot) where it
-- lives, xmin = the transaction that wrote it, xmax = the one that ended it.
SELECT id, balance, ctid, xmin = pg_current_xact_id()::xid AS written_by_me, xmax
FROM accounts ORDER BY id;

-- UPDATE never overwrites: it writes a new version and sets xmax on the old.
UPDATE accounts SET balance = balance - 40 WHERE id = 'a';
SELECT id, balance, ctid, xmax FROM accounts ORDER BY id;

-- A savepoint opens a subtransaction that can be undone on its own.
SAVEPOINT fee;
UPDATE accounts SET balance = balance - 5 WHERE id = 'a';
SELECT id, balance, ctid FROM accounts WHERE id = 'a';
ROLLBACK TO SAVEPOINT fee;

-- Nothing was copied back. Slot 3 is current again, and its xmax still
-- names the subtransaction that was rolled back.
SELECT id, balance, ctid, xmax <> '0'::xid AS xmax_set FROM accounts WHERE id = 'a';

-- The aborted version still occupies slot 4, so the next one goes to slot 5.
UPDATE accounts SET balance = balance - 1 WHERE id = 'a' RETURNING id, balance, ctid;
