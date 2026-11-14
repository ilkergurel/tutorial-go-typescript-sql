-- LEARN(fullstack.tracelite.step-02-schema): the first two tables of migrations/001_init.sql, verbatim.
-- +goose Up
CREATE TABLE projects (
  id         uuid PRIMARY KEY DEFAULT uuidv7(),
  name       text NOT NULL CONSTRAINT projects_name_length CHECK (char_length(name) BETWEEN 1 AND 200),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE requirements (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  project_id  uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
  parent_id   uuid,  -- NULL: top level
  code        text COLLATE "C" NOT NULL CONSTRAINT requirements_code_format
                CHECK (code ~ '^[A-Z][A-Z0-9]*(-[A-Z0-9]+)*$' AND char_length(code) <= 40),
  title       text NOT NULL CONSTRAINT requirements_title_length CHECK (char_length(title) BETWEEN 1 AND 200),
  description text NOT NULL DEFAULT '',
  status      text NOT NULL DEFAULT 'draft'
                CONSTRAINT requirements_status_valid CHECK (status IN ('draft', 'approved', 'obsolete')),
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT requirements_code_key UNIQUE (project_id, code),
  CONSTRAINT requirements_in_project UNIQUE (project_id, id),  -- target of same-project FKs
  CONSTRAINT requirements_not_own_parent CHECK (parent_id <> id),
  CONSTRAINT requirements_parent_fkey FOREIGN KEY (project_id, parent_id)
    REFERENCES requirements (project_id, id)
);
CREATE INDEX requirements_parent ON requirements (parent_id);
-- (test_cases, trace_links and test_runs follow; example 3 runs them)

-- add() inserts by project name and parent code, and reports what a Go store
-- would see: the SQLSTATE and the name of the constraint that refused the row.
CREATE FUNCTION add(project text, parent text, new_code text, new_title text) RETURNS text
LANGUAGE plpgsql AS $$
DECLARE refused_by text;
BEGIN
  INSERT INTO requirements (project_id, parent_id, code, title)
  SELECT p.id, (SELECT r.id FROM requirements r WHERE r.code = parent), new_code, new_title
  FROM projects p WHERE p.name = project;
  RETURN 'ok';
EXCEPTION WHEN others THEN  -- the failed INSERT is rolled back, the script goes on
  GET STACKED DIAGNOSTICS refused_by = CONSTRAINT_NAME;
  RETURN SQLSTATE || ' ' || refused_by;
END $$;

INSERT INTO projects (name) VALUES ('Ödeme'), ('Raporlama');
SELECT attempt, add(project, parent, code, title) AS outcome
FROM (VALUES
  ('top level',                'Ödeme',     NULL,         'AUTH',       'Kimlik doğrulama'),
  ('child',                    'Ödeme',     'AUTH',       'AUTH-LOGIN', 'Parola ile giriş'),
  ('lower-case code',          'Ödeme',     'AUTH',       'auth-reset', 'Parola sıfırlama'),
  ('upper-cased in tr locale', 'Ödeme',     'AUTH',       'AUTH-LOGİN', 'Parola ile giriş'),
  ('same code, same project',  'Ödeme',     NULL,         'AUTH',       'Yetki'),
  ('parent in other project',  'Raporlama', 'AUTH-LOGIN', 'EXPORT-CSV', 'CSV indir'),
  ('200 characters, 400 bytes', 'Ödeme',    NULL,         'LONG',       repeat('ş', 200)),
  ('201 characters',           'Ödeme',     NULL,         'LONGER',     repeat('ş', 201)),
  ('same code, other project', 'Raporlama', NULL,         'AUTH',       'Rapor yetkisi')
) AS t (attempt, project, parent, code, title);

SELECT p.name, r.code, r.parent_id IS NULL AS top_level, octet_length(r.title) AS title_bytes
FROM requirements r JOIN projects p ON p.id = r.project_id
ORDER BY p.name COLLATE "C", r.code;  -- byte order: Ö (U+00D6) sorts after R
