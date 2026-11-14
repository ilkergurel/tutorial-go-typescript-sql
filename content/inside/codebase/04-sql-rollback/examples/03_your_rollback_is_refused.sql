-- Runtime sql: the sandbox is one transaction, so ending it yourself would end
-- the sandbox. sqlrun refuses COMMIT, ROLLBACK, END and ABORT here.
CREATE TABLE drafts (id int);

-- Savepoints rewind inside the one transaction, so they stay allowed.
SAVEPOINT before_drop;
DROP TABLE drafts;
ROLLBACK TO SAVEPOINT before_drop;
SELECT to_regclass('drafts') IS NOT NULL AS drafts_is_back;

-- Refused before it reaches the server; the run stops here.
ROLLBACK;
SELECT 'never runs';
