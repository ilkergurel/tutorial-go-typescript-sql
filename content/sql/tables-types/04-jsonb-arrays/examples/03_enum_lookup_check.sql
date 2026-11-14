-- Option 1, enum: a real type whose values sort in declaration order.
CREATE TYPE ticket_status AS ENUM ('open', 'in_progress', 'done');

-- Option 2, lookup table: values are rows, so they can carry data.
CREATE TABLE statuses (
  code     text PRIMARY KEY,
  label_tr text NOT NULL,
  position int  NOT NULL UNIQUE
);
INSERT INTO statuses VALUES ('open', 'Açık', 10), ('in_progress', 'Sürüyor', 20), ('done', 'Bitti', 30);

CREATE TABLE tickets (
  id        int PRIMARY KEY,
  as_enum   ticket_status NOT NULL,
  as_lookup text NOT NULL REFERENCES statuses,
  -- Option 3, CHECK: plain text, the list lives in the table definition.
  as_check  text NOT NULL CHECK (as_check IN ('open', 'in_progress', 'done'))
);
INSERT INTO tickets VALUES
  (1, 'done', 'done', 'done'),
  (2, 'open', 'open', 'open'),
  (3, 'in_progress', 'in_progress', 'in_progress');

-- enum sorts by declaration, text alphabetically.
SELECT array_agg(as_enum ORDER BY as_enum)   AS enum_order,
       array_agg(as_check ORDER BY as_check) AS text_order
FROM tickets;

-- The lookup table supplies a Turkish label and its own sort order.
SELECT t.id, s.label_tr
FROM tickets t JOIN statuses s ON s.code = t.as_lookup
ORDER BY s.position;

-- Adding a value: one statement each for enum and lookup; CHECK is replaced.
ALTER TYPE ticket_status ADD VALUE 'blocked' BEFORE 'done';
INSERT INTO statuses VALUES ('blocked', 'Engellendi', 25);
ALTER TABLE tickets
  DROP CONSTRAINT tickets_as_check_check,
  ADD CONSTRAINT tickets_as_check_check
    CHECK (as_check IN ('open', 'in_progress', 'blocked', 'done'));

-- Removing a value: an enum has no DROP VALUE at all; the lookup table
-- refuses while rows still use it.
DELETE FROM statuses WHERE code = 'open';
