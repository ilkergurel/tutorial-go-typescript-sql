-- LEARN(sql.querying.null-logic): three-valued logic in one result set.

CREATE TABLE users (name text, email text);
INSERT INTO users VALUES
  ('ada', 'ada@example.com'),
  ('brin', NULL),
  ('cay', 'cay@example.com');

-- 1. Plain inequality: NULL rows silently vanish.
SELECT name FROM users WHERE email <> 'cay@example.com';

-- 2. The null-safe inequality: brin appears.
SELECT name FROM users WHERE email IS DISTINCT FROM 'cay@example.com';

-- 3. Display default for a missing value.
SELECT name, COALESCE(email, '(no email)') AS email FROM users ORDER BY name;
