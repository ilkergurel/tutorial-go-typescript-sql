CREATE SCHEMA tracelite;
CREATE TABLE tracelite.projects (
  id   int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name text NOT NULL
);
CREATE FUNCTION tracelite.project_count() RETURNS bigint
  LANGUAGE sql AS 'SELECT count(*) FROM tracelite.projects';

-- What PUBLIC gets on new objects, before anyone grants anything.
SELECT has_schema_privilege('public', 'tracelite', 'USAGE') AS schema_usage,
       has_table_privilege('public', 'tracelite.projects', 'SELECT') AS table_select,
       has_function_privilege('public', 'tracelite.project_count()', 'EXECUTE') AS function_execute;

-- Change the defaults for objects this role creates from now on.
-- PUBLIC stands in for an app role, which this sandbox cannot create.
ALTER DEFAULT PRIVILEGES IN SCHEMA tracelite
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO PUBLIC;
ALTER DEFAULT PRIVILEGES REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
SELECT pg_get_userbyid(defaclrole) AS for_role,
       CASE defaclnamespace WHEN 0 THEN '(every schema)' ELSE defaclnamespace::regnamespace::text END AS in_schema,
       defaclobjtype AS kind, defaclacl
FROM pg_default_acl ORDER BY kind;

CREATE TABLE tracelite.requirements (
  id    int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  title text NOT NULL
);
CREATE TABLE tracelite.legacy_notes (id serial PRIMARY KEY, body text NOT NULL);
CREATE FUNCTION tracelite.requirement_count() RETURNS bigint
  LANGUAGE sql AS 'SELECT count(*) FROM tracelite.requirements';

SELECT obj, has_table_privilege('public', obj, 'INSERT') AS public_insert
FROM unnest(ARRAY['tracelite.projects', 'tracelite.requirements']) AS obj ORDER BY obj;
SELECT has_function_privilege('public', 'tracelite.requirement_count()', 'EXECUTE') AS function_execute;

-- Default privileges on TABLES don't cover sequences. Simulate an app role
-- that holds table grants only, by revoking our own sequence privileges.
REVOKE ALL ON SEQUENCE tracelite.requirements_id_seq, tracelite.legacy_notes_id_seq FROM playground;
INSERT INTO tracelite.requirements (title) VALUES ('Brake response under 150 ms');
INSERT INTO tracelite.legacy_notes (body) VALUES ('imported from the old tracker');
