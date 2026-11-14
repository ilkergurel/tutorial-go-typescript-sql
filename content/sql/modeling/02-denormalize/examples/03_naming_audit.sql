-- Conventions: snake_case ASCII names, plural tables, id as the primary
-- key, <singular>_id for foreign keys, audit columns on every table.
CREATE TABLE projects (
  id   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name text NOT NULL
);
CREATE TABLE test_cases (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id bigint NOT NULL REFERENCES projects,
  code       text   NOT NULL,
  title      text   NOT NULL CHECK (title <> ''),
  UNIQUE (project_id, code),
  created_at timestamptz NOT NULL DEFAULT now(),
  created_by text        NOT NULL DEFAULT current_user,
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- The names PostgreSQL generated: <table>_<columns>_<kind>.
SELECT conname, contype
FROM pg_constraint
WHERE conrelid = 'test_cases'::regclass AND contype <> 'n'
ORDER BY conname;

INSERT INTO projects (name) VALUES ('Checkout');
INSERT INTO test_cases (project_id, code, title) VALUES (1, 'TC-1', 'Valid card');
UPDATE test_cases SET title = 'Valid Visa card' WHERE code = 'TC-1';

-- A DEFAULT fires on INSERT only, so updated_at didn't move.
SELECT code, title, created_by, updated_at = created_at AS updated_at_unchanged
FROM test_cases;

-- Unquoted names fold to lower case, but only their ASCII letters.
CREATE TABLE "TestRuns" (id int);
CREATE TABLE Ürünler (Fiyat int);
SELECT c.relname, a.attname
FROM pg_class c
JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = 1
WHERE c.relnamespace = current_schema()::regnamespace
  AND c.relname IN ('TestRuns', 'Ürünler')
ORDER BY c.relname;

SELECT * FROM ürünler;
