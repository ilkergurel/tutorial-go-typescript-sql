-- The modern way: an identity column (SQL standard, PostgreSQL 10+).
CREATE TABLE tickets (
  id    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  title text NOT NULL
);

INSERT INTO tickets (title) VALUES ('Login fails'), ('Typo on pricing page')
RETURNING id, title;

-- Sequence values are never handed back: a rolled-back insert leaves a gap.
SAVEPOINT before_insert;
INSERT INTO tickets (title) VALUES ('Rolled back');
ROLLBACK TO SAVEPOINT before_insert;

INSERT INTO tickets (title) VALUES ('After the rollback')
RETURNING id, title;

-- It is still backed by a sequence, one the column owns and manages.
SELECT pg_get_serial_sequence('tickets', 'id')::regclass AS backing_sequence;

-- GENERATED ALWAYS refuses a hand-written id unless you say
-- OVERRIDING SYSTEM VALUE, so accidents can't happen quietly.
INSERT INTO tickets (id, title) VALUES (10, 'Imported by hand');
