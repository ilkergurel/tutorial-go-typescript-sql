-- LEARN(sql.modifying.soft-delete): a deleted_at flag instead of DELETE, and what it breaks.
CREATE TABLE users (
  id         int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  email      text NOT NULL UNIQUE,
  deleted_at timestamptz            -- NULL means live
);
INSERT INTO users (email) VALUES ('selin@example.com'), ('umut@example.com');

-- "Delete" Selin: an UPDATE. The row, and her data, stay.
UPDATE users SET deleted_at = '2026-11-01 10:00+03'
WHERE email = 'selin@example.com';

-- Every query must now remember the filter. Forget it once and a
-- deleted account is back in a list, an export, a count.
SELECT email FROM users ORDER BY id;
SELECT email FROM users WHERE deleted_at IS NULL ORDER BY id;

-- Selin signs up again. The UNIQUE constraint can't tell her deleted
-- row from a live one.
INSERT INTO users (email) VALUES ('selin@example.com');
