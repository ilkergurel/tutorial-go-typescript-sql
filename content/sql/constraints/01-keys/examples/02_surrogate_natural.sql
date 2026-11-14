-- Natural key: an ISO 4217 code is stable, short and meaningful on its own.
CREATE TABLE currencies (
  code text PRIMARY KEY,
  name text NOT NULL
);
INSERT INTO currencies VALUES ('TRY', 'Turkish lira'), ('EUR', 'Euro');

-- Surrogate key alone: every row is "unique", the duplicates included.
CREATE TABLE users_loose (
  id    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  email text NOT NULL
);
INSERT INTO users_loose (email) VALUES ('ayse@example.com'), ('ayse@example.com');
SELECT id, email FROM users_loose ORDER BY id;

-- Surrogate key for references, and the natural key still declared unique.
CREATE TABLE users (
  id       bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  email    text NOT NULL,
  currency text NOT NULL DEFAULT 'TRY' REFERENCES currencies
);
-- Case-insensitive uniqueness is an expression, so it takes a unique index.
CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));

INSERT INTO users (email) VALUES ('ayse@example.com'), ('mehmet@example.com');
-- People change e-mail addresses; rows that reference users.id never notice.
UPDATE users SET email = 'ayse.yilmaz@example.com' WHERE id = 1;
SELECT id, email, currency FROM users ORDER BY id;

INSERT INTO users (email) VALUES ('Ayse.Yilmaz@Example.com');
