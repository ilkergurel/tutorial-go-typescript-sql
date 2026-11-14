-- What the server has on disk, and what a non-superuser may load:
-- trusted = any role with CREATE on the database; otherwise superuser only.
SELECT v.name, v.trusted
FROM pg_available_extension_versions v
JOIN pg_available_extensions a ON a.name = v.name AND a.default_version = v.version
WHERE v.name IN ('pg_trgm', 'unaccent', 'btree_gist', 'tcn',
                 'pg_stat_statements', 'file_fdw', 'postgres_fdw')
ORDER BY v.trusted DESC, v.name;

-- Loading one runs its install script in this database, creating its
-- objects in the first schema on search_path (this run's own schema).
CREATE EXTENSION pg_trgm;
CREATE EXTENSION unaccent;

SELECT e.extname,
       e.extowner::regrole AS owner,
       e.extnamespace::regnamespace::text = current_schema() AS in_this_schema,
       (SELECT count(*) FROM pg_depend d
        WHERE d.refclassid = 'pg_extension'::regclass
          AND d.refobjid = e.oid AND d.deptype = 'e') AS member_objects
FROM pg_extension e
WHERE e.extname IN ('pg_trgm', 'unaccent')
ORDER BY e.extname;

-- New functions, operators and index support, usable at once.
CREATE TABLE requirements (code text PRIMARY KEY, title text NOT NULL);
INSERT INTO requirements VALUES
  ('REQ-1', 'Kullanıcı girişi şifre ile'),
  ('REQ-2', 'Şifre sıfırlama e-postası'),
  ('REQ-3', 'Raporu CSV olarak dışa aktar');
SELECT code, unaccent(title) AS plain,
       round(similarity(unaccent(lower(title)), 'sifre sifirlama')::numeric, 2) AS similarity
FROM requirements
ORDER BY similarity DESC, code;

-- gin_trgm_ops makes LIKE '%...%' indexable, which a B-tree can't do.
CREATE INDEX requirements_title_trgm ON requirements USING gin (title gin_trgm_ops);
SET LOCAL enable_seqscan = off;  -- three rows: make the planner show the index
EXPLAIN (COSTS OFF) SELECT code FROM requirements WHERE title ILIKE '%şifre%';

-- Untrusted: file_fdw reads files on the database server.
CREATE EXTENSION file_fdw;
