-- LEARN(sql.constraints.keys): invariants declared once, enforced for every client.
CREATE TABLE tickets (
  id       bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  title    text NOT NULL,
  status   text NOT NULL DEFAULT 'open',
  priority int  NOT NULL DEFAULT 3,
  assignee text              -- nullable on purpose: "unassigned" is a real state
);

-- An omitted column gets its DEFAULT, and so does the DEFAULT keyword.
INSERT INTO tickets (title) VALUES ('Login fails for İlker');
INSERT INTO tickets (title, status, priority) VALUES ('Export is slow', DEFAULT, 1);

SELECT id, title, status, priority, assignee FROM tickets ORDER BY id;

-- An explicit NULL is a value, not "use the default": NOT NULL rejects it.
INSERT INTO tickets (title, status) VALUES ('Saved by an ORM', NULL);
