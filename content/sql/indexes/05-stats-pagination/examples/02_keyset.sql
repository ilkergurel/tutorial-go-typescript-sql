SET LOCAL TimeZone = 'Europe/Istanbul';
CREATE TABLE posts (
  id         bigint PRIMARY KEY,
  created_at timestamptz NOT NULL,
  title      text NOT NULL
);
-- Two posts per minute, so created_at alone has ties; id breaks them.
INSERT INTO posts
SELECT g, timestamptz '2026-01-01 00:00+03' + (g / 2) * interval '1 minute', 'gönderi ' || g
FROM generate_series(1, 20000) AS g;
CREATE INDEX posts_feed ON posts (created_at, id);
ANALYZE posts;

-- Page 1,601 of a newest-first feed, 5 posts per page, by OFFSET...
SELECT id, created_at FROM posts
ORDER BY created_at DESC, id DESC LIMIT 5 OFFSET 8000;

-- ...and by keyset: "older than the last post on the previous page".
SELECT id, created_at FROM posts
WHERE (created_at, id) < ('2026-01-05 04:00+03', 12001)
ORDER BY created_at DESC, id DESC LIMIT 5;

-- Same rows, very different work.
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT id, created_at FROM posts
ORDER BY created_at DESC, id DESC LIMIT 5 OFFSET 8000;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT id, created_at FROM posts
WHERE (created_at, id) < ('2026-01-05 04:00+03', 12001)
ORDER BY created_at DESC, id DESC LIMIT 5;

-- A reader sees page 1; a new post arrives; the reader asks for page 2.
SELECT id FROM posts ORDER BY created_at DESC, id DESC LIMIT 3;
INSERT INTO posts VALUES (20001, '2026-01-07 22:41+03', 'yeni gönderi');
SELECT id FROM posts ORDER BY created_at DESC, id DESC LIMIT 3 OFFSET 3;
SELECT id FROM posts
WHERE (created_at, id) < ('2026-01-07 22:39+03', 19998)
ORDER BY created_at DESC, id DESC LIMIT 3;
