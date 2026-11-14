-- The legacy way: serial is shorthand for integer + a sequence + a DEFAULT.
CREATE TABLE old_tickets (
  id    serial PRIMARY KEY,
  title text NOT NULL
);

-- Nothing marks it as an identity: it is an ordinary integer column whose
-- default happens to call nextval().
SELECT data_type, column_default, is_identity
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND table_name = 'old_tickets' AND column_name = 'id';

-- An import writes ids by hand, and serial accepts them without a word...
INSERT INTO old_tickets (id, title) VALUES (1, 'Imported'), (2, 'Imported too');

-- ...but the sequence never saw them, so the next default is 1 again.
INSERT INTO old_tickets (title) VALUES ('First new ticket');
