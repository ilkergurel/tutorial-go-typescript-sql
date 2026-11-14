-- One project's requirements, abridged: the project already has AUTH.
CREATE TABLE requirements (
  id        uuid PRIMARY KEY DEFAULT uuidv7(),
  parent_id uuid REFERENCES requirements,
  code      text COLLATE "C" NOT NULL UNIQUE,
  title     text NOT NULL
);
INSERT INTO requirements (code, title) VALUES ('AUTH', 'Kimlik doğrulama');

-- The staging table, as example 3's COPY filled it.
CREATE TEMP TABLE import_rows (
  id     uuid NOT NULL DEFAULT uuidv7(), -- minted before the requirement exists
  line   integer NOT NULL,
  code   text COLLATE "C" NOT NULL,
  title  text NOT NULL,
  parent text COLLATE "C" NOT NULL
) ON COMMIT DROP;
INSERT INTO import_rows (line, code, title, parent) VALUES
  (2, 'AUTH-LOGIN-2FA', 'İki adımlı doğrulama', 'AUTH-LOGIN'),
  (3, 'AUTH-LOGIN', 'Parola ile giriş', 'AUTH'),
  (4, 'AUTH', 'Kimlik doğrulama', ''),
  (5, 'AUTH-LOGIN', 'Parola', 'AUTH'),
  (6, 'EXPORT-CSV', 'CSV indir', 'EXPORT'),
  (7, 'LOOP-A', 'Döngü A', 'LOOP-B'),
  (8, 'LOOP-B', 'Döngü B', 'LOOP-A');

-- The checks no single row can make, as one query over the whole file.
WITH RECURSIVE reached AS (
  SELECT code FROM import_rows
  WHERE parent = '' OR parent NOT IN (SELECT code FROM import_rows)
  UNION
  SELECT s.code FROM import_rows s JOIN reached r ON s.parent = r.code
)
SELECT line, 'code' AS "column", 'already used in this project' AS detail
FROM import_rows s
WHERE EXISTS (SELECT FROM requirements r WHERE r.code = s.code)
UNION ALL
SELECT line, 'code', 'also on line ' || first
FROM (SELECT line, min(line) OVER (PARTITION BY code) AS first
      FROM import_rows) AS d
WHERE line > first
UNION ALL
SELECT line, 'parent', 'no requirement ' || parent || ' in the file or the project'
FROM import_rows s
WHERE parent <> '' AND parent NOT IN (SELECT code FROM import_rows)
  AND NOT EXISTS (SELECT FROM requirements r WHERE r.code = s.parent)
UNION ALL
SELECT line, 'parent', 'its chain of parents loops and never reaches the top'
FROM import_rows
WHERE code NOT IN (SELECT code FROM reached)
ORDER BY 1, 2;

-- The fixed file keeps lines 2 and 3. One INSERT creates the tree, although
-- the child comes before its parent.
DELETE FROM import_rows WHERE line > 3;
INSERT INTO requirements (id, parent_id, code, title)
SELECT s.id, coalesce(p.id, e.id), s.code, s.title
FROM import_rows s
LEFT JOIN import_rows p ON p.code = s.parent
LEFT JOIN requirements e ON e.code = s.parent;

SELECT r.code, p.code AS parent, r.title
FROM requirements r LEFT JOIN requirements p ON p.id = r.parent_id
ORDER BY r.code;
