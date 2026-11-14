-- A requirement can refine a parent requirement in the same table.
CREATE TABLE requirements (
  id        int PRIMARY KEY,
  parent_id int REFERENCES requirements,
  code      text NOT NULL,
  title     text NOT NULL
);
INSERT INTO requirements VALUES
  (1, NULL, 'REQ-1',   'Accounts'),
  (2, 1,    'REQ-1.1', 'Kayıt formu'),
  (3, 1,    'REQ-1.2', 'KAYIT FORMU'),
  (4, NULL, 'REQ-2',   'Reports'),
  (5, 4,    'REQ-2.1', 'Export csv'),
  (6, 4,    'REQ-2.2', 'Export CSV');

-- Two aliases, one table: each row next to its parent. LEFT keeps the roots.
SELECT c.code, c.title, p.code AS parent
FROM requirements c
LEFT JOIN requirements p ON p.id = c.parent_id
ORDER BY c.id;

-- Pairs within one table: a.id < b.id drops self-pairs and mirror images.
SELECT a.code, b.code AS same_title_as, a.title
FROM requirements a
JOIN requirements b ON lower(a.title) = lower(b.title) AND a.id < b.id
ORDER BY a.id;

-- The database's C.UTF-8 rules lower 'KAYIT' to 'kayit'. Turkish rules:
SELECT a.code, b.code AS same_title_as, a.title
FROM requirements a
JOIN requirements b
  ON lower(a.title COLLATE "tr-x-icu") = lower(b.title COLLATE "tr-x-icu")
 AND a.id < b.id
ORDER BY a.id;
