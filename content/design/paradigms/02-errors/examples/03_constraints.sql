-- LEARN(design.paradigms.errors): in SQL the failure modes are declared constraints; a refused write is a SQLSTATE.
-- A: and B: are two connections. Outside BEGIN, each statement commits on its own.
A: CREATE TABLE users (
     id    int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
     email text NOT NULL UNIQUE CHECK (email LIKE '%@%')
   );
A: INSERT INTO users (email) VALUES ('ayse@example.com');

-- The three sign-ups from the Go and TS examples.
A: INSERT INTO users (email) VALUES ('mehmet@example.com');
A: INSERT INTO users (email) VALUES ('ayse@example.com');
A: INSERT INTO users (email) VALUES ('can.example.com');

-- The race an application check can't close: both see the address as free.
A: BEGIN;
B: BEGIN;
A: SELECT count(*) FROM users WHERE email = 'zeynep@example.com';
B: SELECT count(*) FROM users WHERE email = 'zeynep@example.com';
A: INSERT INTO users (email) VALUES ('zeynep@example.com');
B: INSERT INTO users (email) VALUES ('zeynep@example.com');
A: COMMIT;

-- B's error aborted its whole transaction, like an exception unwinding it.
B: SELECT count(*) FROM users;
B: ROLLBACK;

A: SELECT id, email FROM users ORDER BY id;
