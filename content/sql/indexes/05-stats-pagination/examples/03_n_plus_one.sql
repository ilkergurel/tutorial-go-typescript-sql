CREATE TABLE posts (id int PRIMARY KEY, title text NOT NULL);
CREATE TABLE comments (
  id      int PRIMARY KEY,
  post_id int NOT NULL REFERENCES posts,
  body    text NOT NULL
);
CREATE INDEX comments_post ON comments (post_id);
INSERT INTO posts SELECT g, 'gönderi ' || g FROM generate_series(1, 1000) AS g;
INSERT INTO comments SELECT g, g % 1000 + 1, 'yorum ' || g FROM generate_series(1, 10000) AS g;
ANALYZE posts, comments;

-- N+1, the way lazy loading sends it: one query for the page...
SELECT id, title FROM posts ORDER BY id LIMIT 3;
-- ...then one more query per row: N more round trips, parses and plans.
SELECT count(*) FROM comments WHERE post_id = 1;
SELECT count(*) FROM comments WHERE post_id = 2;
SELECT count(*) FROM comments WHERE post_id = 3;

-- Batched: one statement for the whole page, the ids as one array parameter.
PREPARE comment_counts(int[]) AS
  SELECT post_id, count(*) AS comments
  FROM comments WHERE post_id = ANY ($1)
  GROUP BY post_id ORDER BY post_id;
EXECUTE comment_counts('{1,2,3}');

-- Or the page and its counts in a single query.
SELECT p.id, p.title, count(c.id) AS comments
FROM (SELECT id, title FROM posts ORDER BY id LIMIT 3) AS p
LEFT JOIN comments c ON c.post_id = p.id
GROUP BY p.id, p.title
ORDER BY p.id;

-- The per-row shape inside one statement: a SubPlan that runs once per row.
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT p.id, (SELECT count(*) FROM comments c WHERE c.post_id = p.id) AS comments
FROM posts p ORDER BY p.id LIMIT 3;
