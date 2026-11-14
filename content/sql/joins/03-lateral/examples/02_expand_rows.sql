CREATE TABLE requirements (
  id   int  PRIMARY KEY,
  code text NOT NULL,
  tags text[] NOT NULL DEFAULT '{}'
);
INSERT INTO requirements VALUES
  (1, 'REQ-1', '{security,login}'),
  (2, 'REQ-2', '{export}'),
  (3, 'REQ-3', '{}');

-- A function in FROM is implicitly LATERAL: it takes r.tags as its input and
-- returns one row per element. REQ-3 has no elements, so it drops out.
SELECT r.code, tag.name, tag.pos
FROM requirements r
CROSS JOIN LATERAL unnest(r.tags) WITH ORDINALITY AS tag(name, pos)
ORDER BY r.code, tag.pos;

-- LEFT JOIN ... ON true keeps it, with NULLs.
SELECT r.code, tag.name
FROM requirements r
LEFT JOIN LATERAL unnest(r.tags) AS tag(name) ON true
ORDER BY r.code, tag.name;

-- One row per day of each sprint, generated from that row's own columns.
CREATE TABLE sprints (name text PRIMARY KEY, starts date NOT NULL, ends date NOT NULL);
INSERT INTO sprints VALUES ('S1', '2026-11-02', '2026-11-04'), ('S2', '2026-11-05', '2026-11-06');

SELECT s.name, d::date AS day
FROM sprints s
CROSS JOIN LATERAL generate_series(s.starts, s.ends, interval '1 day') AS d
ORDER BY d;
