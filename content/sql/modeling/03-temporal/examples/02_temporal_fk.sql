CREATE EXTENSION btree_gist;

-- Project versions over time. Project 1 was renamed on 1 July, so it has
-- two rows; project 2 was closed at the end of March.
CREATE TABLE projects (
  id       int,
  name     text NOT NULL,
  valid_at daterange,
  PRIMARY KEY (id, valid_at WITHOUT OVERLAPS)
);
INSERT INTO projects VALUES
  (1, 'Checkout',      '[2026-01-01,2026-07-01)'),
  (1, 'Payments',      '[2026-07-01,)'),
  (2, 'Legacy import', '[2026-01-01,2026-04-01)');

-- A membership must lie within its project's lifetime: a temporal FK.
CREATE TABLE memberships (
  project_id int,
  member     text,
  valid_at   daterange,
  PRIMARY KEY (project_id, member, valid_at WITHOUT OVERLAPS),
  FOREIGN KEY (project_id, PERIOD valid_at)
    REFERENCES projects (id, PERIOD valid_at)
);
INSERT INTO memberships VALUES
  -- Spans the rename: neither version covers it alone, both together do.
  (1, 'Ayşe', '[2026-03-01,2026-09-01)'),
  (2, 'Can',  '[2026-02-01,2026-04-01)');

-- A temporal join: match on the key and on overlapping periods, and
-- keep the intersection (*) as the period the combined row is true for.
SELECT m.member, p.name AS project, m.valid_at * p.valid_at AS during
FROM memberships m
JOIN projects p ON p.id = m.project_id AND p.valid_at && m.valid_at
ORDER BY m.member, lower(m.valid_at * p.valid_at);

-- Can stays on project 2 after it closed: nothing covers April to June.
INSERT INTO memberships VALUES (2, 'Can', '[2026-04-01,2026-07-01)');
