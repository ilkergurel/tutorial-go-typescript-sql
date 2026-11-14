CREATE TABLE users (
  id         int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  email      text NOT NULL,
  deleted_at timestamptz
);
-- Unique among live rows only: deleted rows are not in this index.
CREATE UNIQUE INDEX users_email_live ON users (email) WHERE deleted_at IS NULL;

-- One place that knows what "live" means; app queries read from it.
CREATE VIEW live_users AS
  SELECT id, email FROM users WHERE deleted_at IS NULL;

INSERT INTO users (email) VALUES ('selin@example.com'), ('umut@example.com');
UPDATE users SET deleted_at = '2026-11-01 10:00+03' WHERE id = 1;

-- Re-signup works now: the old row is outside the index predicate.
INSERT INTO users (email) VALUES ('selin@example.com');

SELECT id, email, deleted_at IS NOT NULL AS deleted FROM users ORDER BY id;
SELECT id, email FROM live_users ORDER BY id;

-- Two LIVE rows with one email are still impossible.
INSERT INTO users (email) VALUES ('umut@example.com');
